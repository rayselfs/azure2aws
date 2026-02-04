# End-to-End Testing Guide

This document outlines the end-to-end testing strategy for azure2aws.

## Prerequisites

Before running tests, ensure you have:
- An Azure AD tenant with SAML configured for AWS
- Valid Azure AD credentials (username/password)
- MFA device configured and accessible
- AWS account accessible via SAML

## Manual Testing Workflow

### 1. Configure Profile

```bash
./bin/azure2aws configure --profile test
```

**Expected prompts:**
- Azure AD MyApps URL
- App ID (GUID)
- Username (email)
- Optional: AWS Role ARN

**Verify:**
- Config file created at `~/.azure2aws/config.yaml`
- File permissions are `0600`
- Profile appears in config file

### 2. Login Flow

```bash
./bin/azure2aws login --profile test
```

**Expected behavior:**
- Prompts for password (unless stored in keyring)
- Handles Azure AD authentication
- Handles MFA challenge automatically
- Parses SAML response
- Lists available AWS roles (if multiple)
- Calls STS AssumeRoleWithSAML
- Saves credentials to `~/.aws/credentials`
- Offers to save password to keyring

**Verify:**
- Credentials saved in `~/.aws/credentials` under [test] section
- File permissions are `0600`
- Credentials include:
  - `aws_access_key_id`
  - `aws_secret_access_key`
  - `aws_session_token`
  - `x_security_token_expires` (timestamp)

### 3. Credentials Validation

```bash
aws --profile test sts get-caller-identity
```

**Expected output:**
```json
{
    "UserId": "AROA...:user@example.com",
    "Account": "123456789012",
    "Arn": "arn:aws:sts::123456789012:assumed-role/RoleName/user@example.com"
}
```

### 4. Exec Command

```bash
./bin/azure2aws exec --profile test -- aws s3 ls
```

**Expected behavior:**
- Loads credentials from `~/.aws/credentials`
- Validates expiration
- Sets environment variables
- Executes subprocess
- Displays S3 buckets

**Verify environment variables:**
```bash
./bin/azure2aws exec --profile test -- env | grep AWS
```

Should include:
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_SESSION_TOKEN`
- `AWS_REGION` (if configured)
- `AWS_CREDENTIAL_EXPIRATION`

### 5. Console Command

```bash
./bin/azure2aws console --profile test
```

**Expected behavior:**
- Loads credentials
- Calls AWS Federation endpoint
- Receives signin token
- Opens browser to AWS Console
- User is automatically logged in

**Test link-only mode:**
```bash
./bin/azure2aws console --profile test --link
```

Should print federation URL without opening browser.

**Test service navigation:**
```bash
./bin/azure2aws console --profile test --service ec2
```

Should open directly to EC2 console.

### 6. Credential Expiration

Wait for credentials to expire (or manually modify expiration in `~/.aws/credentials`).

```bash
./bin/azure2aws exec --profile test -- aws s3 ls
```

**Expected behavior:**
- Detects expired credentials
- Returns error message
- Suggests running `azure2aws login --profile test`

### 7. Force Re-authentication

```bash
./bin/azure2aws login --profile test --force
```

**Expected behavior:**
- Bypasses credential validity check
- Forces new authentication
- Updates credentials

### 8. Verbose and Debug Modes

```bash
./bin/azure2aws login --profile test --verbose
./bin/azure2aws login --profile test --debug
```

**Expected behavior:**
- Verbose: Shows progress messages
- Debug: Shows detailed HTTP requests/responses (with sensitive data redacted)

## Security Testing

### 1. Sensitive Data Redaction

Check logs for redaction of sensitive data:
```bash
./bin/azure2aws login --profile test --debug 2>&1 | grep -i password
```

Should show `[REDACTED]` instead of actual passwords.

### 2. File Permissions

```bash
ls -la ~/.azure2aws/config.yaml
ls -la ~/.aws/credentials
```

Both should be `-rw-------` (0600 - read/write by owner only).

### 3. TLS Verification

Network traffic should use TLS 1.2+ with proper certificate validation (default behavior).

## Automated Testing

### Unit Tests

```bash
cd azure2aws
go test ./... -v
```

### Build Verification

```bash
make build
make test
make lint
```

### Cross-Platform Build

```bash
GOOS=linux GOARCH=amd64 go build -o bin/azure2aws-linux ./cmd/azure2aws
GOOS=darwin GOARCH=amd64 go build -o bin/azure2aws-darwin ./cmd/azure2aws
GOOS=windows GOARCH=amd64 go build -o bin/azure2aws.exe ./cmd/azure2aws
```

## Error Scenarios

### 1. Invalid Credentials

Test with wrong password:
**Expected:** Authentication failure with clear error message

### 2. MFA Failure

Test with incorrect/timeout MFA:
**Expected:** MFA failure with retry or error message

### 3. Network Failure

Test with no network:
**Expected:** Connection error with descriptive message

### 4. Invalid Configuration

Test with missing/malformed config:
**Expected:** Configuration error with guidance

### 5. Missing AWS Role

Test with profile that has no AWS roles:
**Expected:** Clear error message about missing roles

## Performance Testing

### Login Performance

```bash
time ./bin/azure2aws login --profile test
```

**Expected:** Complete within 10-30 seconds (depending on MFA method)

### Exec Overhead

```bash
time ./bin/azure2aws exec --profile test -- echo "test"
```

**Expected:** Minimal overhead (< 1 second)

## Cleanup

After testing:
```bash
rm ~/.azure2aws/config.yaml
rm -rf ~/.azure2aws/
aws configure --profile test set aws_access_key_id ""
aws configure --profile test set aws_secret_access_key ""
aws configure --profile test set aws_session_token ""
```

## Known Limitations

1. MFA is fixed to "Auto" mode (uses Azure AD default method)
2. Only supports Azure AD provider
3. Requires graphical browser for Console command (use `--link` for headless)

## Continuous Integration

See `.github/workflows/ci.yaml` for automated CI testing including:
- Multi-platform builds (Linux, macOS, Windows)
- Unit tests with race detection
- golangci-lint checks
- Code coverage reporting
