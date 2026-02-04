package cmd

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const (
	githubAPIURL   = "https://api.github.com/repos/rayselfs/azure2aws/releases/latest"
	updateRepoName = "rayselfs/azure2aws"
)

type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func newUpdateCmd(currentVersion string) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update azure2aws to the latest version",
		Long: `Checks for updates and downloads the latest version from GitHub.

The binary is verified using SHA256 checksum before installation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(currentVersion, force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force update even if current version is latest")

	return cmd
}

func runUpdate(currentVersion string, force bool) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	fmt.Println("Checking for updates...")
	release, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !force && release.TagName == currentVersion {
		fmt.Printf("Already running the latest version: %s\n", currentVersion)
		return nil
	}

	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Printf("Latest version:  %s\n", release.TagName)

	asset, checksumAsset := findAssets(release, runtime.GOOS, runtime.GOARCH)
	if asset == nil {
		return fmt.Errorf("no release found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("Downloading %s...\n", asset.Name)
	tmpFile, err := downloadFile(asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer os.Remove(tmpFile)

	if checksumAsset != nil {
		fmt.Println("Verifying checksum...")
		if err := verifyChecksum(tmpFile, asset.Name, checksumAsset.BrowserDownloadURL); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	fmt.Println("Extracting binary...")
	binaryPath, err := extractBinary(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}
	defer os.Remove(binaryPath)

	fmt.Println("Installing update...")
	if err := replaceBinary(execPath, binaryPath); err != nil {
		return fmt.Errorf("failed to install update: %w", err)
	}

	fmt.Printf("Successfully updated to %s\n", release.TagName)
	return nil
}

func getLatestRelease() (*GitHubRelease, error) {
	resp, err := http.Get(githubAPIURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func findAssets(release *GitHubRelease, goos, goarch string) (*GitHubAsset, *GitHubAsset) {
	var asset, checksumAsset *GitHubAsset

	archiveName := fmt.Sprintf("azure2aws_%s_%s_%s.tar.gz", strings.TrimPrefix(release.TagName, "v"), goos, goarch)
	checksumName := "azure2aws_checksums.txt"

	for i := range release.Assets {
		if release.Assets[i].Name == archiveName {
			asset = &release.Assets[i]
		}
		if release.Assets[i].Name == checksumName {
			checksumAsset = &release.Assets[i]
		}
	}

	return asset, checksumAsset
}

func downloadFile(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "azure2aws-update-*")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func verifyChecksum(archivePath, archiveName, checksumURL string) error {
	resp, err := http.Get(checksumURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	checksumData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var expectedChecksum string
	for _, line := range strings.Split(string(checksumData), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == archiveName {
			expectedChecksum = parts[0]
			break
		}
	}

	if expectedChecksum == "" {
		return fmt.Errorf("checksum not found for %s", archiveName)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actualChecksum := hex.EncodeToString(h.Sum(nil))

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

func extractBinary(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if header.Name == "azure2aws" || header.Name == "azure2aws.exe" {
			tmpFile, err := os.CreateTemp("", "azure2aws-new-*")
			if err != nil {
				return "", err
			}
			defer tmpFile.Close()

			if _, err := io.Copy(tmpFile, tr); err != nil {
				os.Remove(tmpFile.Name())
				return "", err
			}

			if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
				os.Remove(tmpFile.Name())
				return "", err
			}

			return tmpFile.Name(), nil
		}
	}

	return "", fmt.Errorf("azure2aws binary not found in archive")
}

func replaceBinary(oldPath, newPath string) error {
	backupPath := oldPath + ".backup"
	if err := os.Rename(oldPath, backupPath); err != nil {
		return err
	}

	if err := copyFile(newPath, oldPath); err != nil {
		os.Rename(backupPath, oldPath)
		return err
	}

	os.Remove(backupPath)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return os.Chmod(dst, 0755)
}
