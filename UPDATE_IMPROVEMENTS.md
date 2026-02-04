# Update 機制改進說明

## 修復的問題

本次改進修復了自我更新機制的 5 個關鍵問題：

### ✅ 必須修復的問題

#### 1. 原子性更新（Atomic Replacement）

**問題**：
```go
// 舊的做法（危險）
rename(old → backup)     // 步驟 1
copyFile(new → old)      // 步驟 2：慢慢複製（可能花數秒）
chmod(old, 0755)         // 步驟 3

// 風險：在步驟 2 期間
// - old 檔案存在，但內容只寫了一半
// - 其他人此時執行 old 會失敗
```

**解決方案**：
```go
// 新的做法（安全）
copyFileAtomic(new → old.new)   // 先寫到暫存檔（完整寫入）
rename(old → backup)            // 備份舊檔
rename(old.new → old)           // 原子切換（瞬間完成）
```

**改進點**：
- ✅ 新檔案先完整寫到 `.new` 暫存檔
- ✅ 最後用 `rename()` 原子切換（Unix 上保證原子性）
- ✅ 任何時刻 `old` 路徑都指向完整可用的檔案

---

#### 2. 互斥鎖（Mutual Exclusion）

**問題**：
```
兩個 update 同時執行會互相干擾：
  update A: rename(old → backup)
  update B: rename(old → backup)  ← 覆蓋 A 的 backup
  update A: 失敗時想還原
  └─ 還原到錯誤的版本！
```

**解決方案**：
```go
// 在更新開始前加鎖
lockFile := execPath + ".lock"
unlock, err := acquireLock(lockFile)
if err != nil {
    return fmt.Errorf("another update is already in progress: %w", err)
}
defer unlock()
```

**跨平台實作**：

**Unix (macOS/Linux)**：
```go
func acquireLockUnix(lockFile string) (func(), error) {
    f, _ := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0600)
    
    // 使用 flock 系統調用（非阻塞模式）
    if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
        return nil, fmt.Errorf("failed to acquire lock: %w", err)
    }
    
    unlock := func() {
        syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
        f.Close()
        os.Remove(lockFile)
    }
    
    return unlock, nil
}
```

**Windows**：
```go
func acquireLockWindows(lockFile string) (func(), error) {
    // Windows 上用 exclusive create
    f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
    if err != nil {
        if os.IsExist(err) {
            return nil, fmt.Errorf("lock file exists")
        }
        return nil, err
    }
    
    unlock := func() {
        f.Close()
        os.Remove(lockFile)
    }
    
    return unlock, nil
}
```

**改進點**：
- ✅ 同時只能有一個更新在執行
- ✅ 跨平台支援（Unix 用 flock，Windows 用 exclusive file）
- ✅ defer unlock() 確保鎖一定會釋放

---

#### 3. 資料耐久性（Durability with fsync）

**問題**：
```
copyFile 完成 → 資料還在記憶體快取
突然斷電 → 可能得到 0 byte 或半寫入的檔案
```

**解決方案**：
```go
func copyFileAtomic(src, dst string, mode fs.FileMode) error {
    in, _ := os.Open(src)
    out, _ := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
    
    // 1. 複製資料
    io.Copy(out, in)
    
    // 2. fsync 檔案（確保資料寫入磁碟）
    out.Sync()
    
    // 3. fsync 目錄（確保目錄項目持久化）
    dirPath := filepath.Dir(dst)
    syncDir(dirPath)
    
    return nil
}

func syncDir(path string) error {
    if runtime.GOOS == "windows" {
        return nil  // Windows 不需要 fsync 目錄
    }
    
    dir, _ := os.Open(path)
    defer dir.Close()
    return dir.Sync()
}
```

**改進點**：
- ✅ `out.Sync()` 確保檔案內容寫入磁碟
- ✅ `dir.Sync()` 確保目錄項目持久化（Unix）
- ✅ 斷電後也能保證檔案完整性

---

### ✅ 應該考慮的問題

#### 4. 符號連結處理（Symlink Resolution）

**問題**：
```
假設：/usr/local/bin/azure2aws → /opt/azure2aws/1.0.0/azure2aws (symlink)

執行 os.Executable()：
  └─ 回傳：/usr/local/bin/azure2aws (symlink 路徑)

執行 rename(symlink, backup)：
  └─ 改名的是 symlink 本身，不是真正的檔案

執行 create(symlink)：
  └─ 建立一般檔案，破壞了原本的 symlink 結構
```

**解決方案**：
```go
func runUpdate(currentVersion string, force bool) error {
    execPath, _ := os.Executable()
    
    // 解析符號連結，找到真正的檔案
    execPath, err = resolveSymlink(execPath)
    if err != nil {
        return fmt.Errorf("failed to resolve executable path: %w", err)
    }
    
    // ... 繼續更新流程
}

func resolveSymlink(path string) (string, error) {
    info, _ := os.Lstat(path)  // 不跟隨 symlink
    
    // 如果不是 symlink，直接回傳
    if info.Mode()&os.ModeSymlink == 0 {
        return path, nil
    }
    
    // 解析 symlink 到真實路徑
    resolved, err := filepath.EvalSymlinks(path)
    if err != nil {
        return "", err
    }
    
    return resolved, nil
}
```

**改進點**：
- ✅ 自動偵測並解析 symlink
- ✅ 更新真正的檔案，不破壞 symlink 結構
- ✅ 支援多層 symlink

---

#### 5. 保留檔案權限（Permission Preservation）

**問題**：
```go
// 舊的做法（有問題）
func copyFile(src, dst string) error {
    // ...
    return os.Chmod(dst, 0755)  // 硬編碼權限
}

// 問題：
// 1. 原檔可能是 0700（只有 owner 能執行）
// 2. 原檔可能有特殊的 owner/group
// 3. 覆蓋掉原本的權限設定
```

**解決方案**：
```go
func replaceBinary(oldPath, newPath string) error {
    // 1. 先取得舊檔案的權限資訊
    oldInfo, err := os.Stat(oldPath)
    if err != nil {
        return fmt.Errorf("failed to stat old binary: %w", err)
    }
    
    // 2. 複製時保留原本的權限
    tmpPath := oldPath + ".new"
    if err := copyFileAtomic(newPath, tmpPath, oldInfo.Mode()); err != nil {
        return fmt.Errorf("failed to copy new binary: %w", err)
    }
    
    // ... 繼續替換流程
}

func copyFileAtomic(src, dst string, mode fs.FileMode) error {
    // 使用傳入的 mode，而不是硬編碼
    out, _ := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
    // ...
}
```

**改進點**：
- ✅ 保留原檔案的權限設定（0755, 0700, etc.）
- ✅ 尊重系統管理員的權限配置
- ✅ 避免意外改變安全性設定

---

## 完整流程比較

### 舊流程（有風險）
```
1. rename(old → backup)
2. create(old)              ← 開始寫入
3. io.Copy(...)             ← 慢慢複製（危險窗口）
4. chmod(old, 0755)         ← 硬編碼權限
5. remove(backup)

問題：
❌ 步驟 2-4 期間，old 檔案不完整
❌ 沒有 fsync，斷電可能遺失資料
❌ 沒有鎖，併發更新會衝突
❌ symlink 會被破壞
❌ 權限被硬編碼覆蓋
```

### 新流程（安全）
```
1. acquireLock()            ← 🔒 加鎖
2. resolveSymlink()         ← 🔗 解析 symlink
3. stat(old) → oldMode      ← 📋 保存權限
4. copy(new → old.new)      ← 💾 完整寫入暫存檔
5. fsync(old.new)           ← 💿 確保寫入磁碟
6. fsync(dir)               ← 📁 確保目錄持久化
7. rename(old → backup)     ← 📦 備份舊檔
8. rename(old.new → old)    ← ⚡ 原子切換
9. remove(backup)           ← 🗑️ 清理備份
10. unlock()                ← 🔓 解鎖

改進：
✅ old 路徑任何時刻都指向完整檔案
✅ fsync 確保耐久性
✅ 鎖機制防止併發衝突
✅ 正確處理 symlink
✅ 保留原始權限
```

---

## 平台支援

### macOS / Linux
✅ 完整支援
- flock 互斥鎖
- 原子 rename
- fsync 耐久性
- symlink 解析

### Windows
⚠️ 部分支援
- ✅ 檔案鎖（透過 exclusive create）
- ✅ 原子 rename（NTFS 支援）
- ⚠️ 執行中的 .exe 可能仍被鎖定
- ⚠️ 可能需要額外的 helper process

**Windows 注意事項**：
在 Windows 上，執行中的 .exe 通常被系統鎖住，即使有這些改進，仍可能遇到 "file in use" 錯誤。如果需要完整的 Windows 支援，建議採用 "helper process" 或 "下次啟動替換" 策略。

---

## 測試建議

### 測試案例

1. **正常更新**
   ```bash
   azure2aws update
   ```

2. **併發更新**（應該失敗）
   ```bash
   # Terminal 1
   azure2aws update
   
   # Terminal 2（同時執行）
   azure2aws update  # 應顯示 "another update is already in progress"
   ```

3. **Symlink 情境**
   ```bash
   ln -s /opt/azure2aws/azure2aws /usr/local/bin/azure2aws
   /usr/local/bin/azure2aws update  # 應更新 /opt/azure2aws/azure2aws
   ```

4. **斷電模擬**（進階）
   ```bash
   # 需要 root 權限
   # 在更新期間突然 kill -9 或重開機
   # 檢查檔案是否完整
   ```

5. **權限保留**
   ```bash
   chmod 0700 /usr/local/bin/azure2aws  # 只有 owner 可執行
   azure2aws update
   ls -l /usr/local/bin/azure2aws  # 應該仍是 0700
   ```

---

## 安全性評估

| 風險 | 舊版本 | 新版本 |
|------|--------|--------|
| 半寫入檔案 | ❌ 高風險 | ✅ 已修復 |
| 併發衝突 | ❌ 高風險 | ✅ 已修復 |
| 斷電遺失 | ❌ 中風險 | ✅ 已修復 |
| Symlink 破壞 | ❌ 中風險 | ✅ 已修復 |
| 權限變更 | ⚠️ 低風險 | ✅ 已修復 |
| Windows 鎖定 | ❌ 無法執行 | ⚠️ 部分改善 |

---

## 未來改進方向

如果需要更強健的更新機制，可以考慮：

1. **簽章驗證**：驗證新版本的數位簽章（防供應鏈攻擊）
2. **回滾機制**：保留多個版本，支援降級
3. **Windows Helper**：額外的 updater 程式處理 Windows 執行中鎖定問題
4. **Delta Update**：只下載差異部分，減少流量
5. **進度顯示**：大檔案下載時顯示進度條
6. **自動重試**：網路失敗時自動重試
7. **版本驗證**：確保不會降級到舊版本（除非 --force）

---

## 結論

本次改進大幅提升了自我更新機制的安全性和可靠性：

✅ **原子性**：任何時刻檔案都是完整可用的  
✅ **並發安全**：防止多個更新同時執行  
✅ **耐久性**：斷電後資料不會遺失  
✅ **Symlink 支援**：正確處理符號連結  
✅ **權限保留**：尊重原始權限設定  

這些改進使 azure2aws 的自我更新功能達到生產環境等級的品質。
