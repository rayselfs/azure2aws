# Azure2AWS Usage Examples

## Configure with Region and Output Format

### Interactive Mode
```bash
# Interactive prompts will ask for region and output
azure2aws configure -p myprofile
```

### Non-Interactive Mode
```bash
# Specify all parameters via flags
azure2aws configure -p myprofile \
  --url "https://myapps.microsoft.com/signin/AWS/app-id" \
  --app-id "12345678-1234-1234-1234-123456789abc" \
  --username "user@example.com" \
  --region "us-west-2" \
  --output "json"
```

### Update Existing Profile
```bash
# Change region and output format for existing profile
azure2aws configure -p production \
  --region "ap-northeast-1" \
  --output "table"
```

## Login and Credential Storage

When you run `azure2aws login`, the tool will:

1. Authenticate with Azure AD using SAML
2. Retrieve temporary AWS credentials from STS
3. Save credentials to `~/.aws/credentials`:
   ```ini
   [myprofile]
   aws_access_key_id = ASIA...
   aws_secret_access_key = ...
   aws_session_token = ...
   x_security_token_expires = 2026-02-04T12:00:00Z
   ```

4. Save configuration to `~/.aws/config`:
   ```ini
   [profile myprofile]
   region = us-west-2
   output = json
   ```

## Using AWS CLI with Credentials

After login, you can use AWS CLI directly:

```bash
# Use specific profile
aws s3 ls --profile myprofile

# Or set as default
export AWS_PROFILE=myprofile
aws ec2 describe-instances
```

## Output Formats

Azure2AWS supports all AWS CLI output formats:

- `json` (default) - JSON format
- `text` - Tab-delimited text
- `table` - ASCII table format
- `yaml` - YAML format (AWS CLI v2)
- `yaml-stream` - YAML stream format (AWS CLI v2)

Example:
```bash
# Configure with table output
azure2aws configure -p dev --output table

# After login, AWS CLI commands will use table format by default
aws ec2 describe-instances --profile dev
```

## Complete Workflow Example

```bash
# 1. Configure profile with region and output
azure2aws configure -p production \
  --url "https://myapps.microsoft.com/signin/AWS/xxx" \
  --app-id "xxx-xxx-xxx" \
  --username "admin@company.com" \
  --region "us-east-1" \
  --output "json"

# 2. Login (will prompt for password or use keyring)
azure2aws login -p production

# 3. Verify credentials
aws sts get-caller-identity --profile production

# 4. Use AWS services
aws s3 ls --profile production
aws ec2 describe-instances --profile production

# 5. Execute commands with credentials
azure2aws exec -p production -- aws s3 ls

# 6. Open AWS Console in browser
azure2aws console -p production
```

## Environment Variables

The tool respects standard AWS environment variables:

- `AWS_SHARED_CREDENTIALS_FILE` - Custom credentials file path
- `AWS_CONFIG_FILE` - Custom config file path
- `AWS_PROFILE` - Default profile to use

Example:
```bash
export AWS_PROFILE=production
azure2aws login  # Uses 'production' profile
```

## Session Duration Configuration

### Setting Session Duration

The default session duration is 3600 seconds (1 hour). You can override this per profile:

```bash
# Configure with custom session duration
azure2aws configure -p prod \
  --url "https://account.activedirectory.windowsazure.com" \
  --app-id "xxx-xxx-xxx" \
  --username "user@example.com" \
  --region "us-east-2" \
  --output "json" \
  --session-duration 3600

# Update only session duration for existing profile
azure2aws configure -p prod --session-duration 7200
```

### Session Duration Values

Common session durations:
- `900` - 15 minutes
- `1800` - 30 minutes
- `3600` - 1 hour (default)
- `7200` - 2 hours
- `14400` - 4 hours
- `28800` - 8 hours
- `43200` - 12 hours (maximum)

**Note**: The actual session duration is limited by:
1. Your AWS IAM Role's "Maximum session duration" setting
2. The duration specified in the SAML assertion from Azure AD

If you encounter the error "DurationSeconds exceeds the MaxSessionDuration", reduce the session duration in your profile configuration.

### Configuration Priority

Session duration is determined in this order:
1. Profile-specific `session_duration` (highest priority)
2. SAML assertion duration
3. Default `session_duration` (3600 seconds)

Example configuration:
```yaml
defaults:
  region: us-east-1
  session_duration: 3600  # Default 1 hour

profiles:
  prod:
    url: https://account.activedirectory.windowsazure.com
    app_id: xxx-xxx-xxx
    username: user@example.com
    region: us-east-2
    output: json
    session_duration: 3600  # Override for this profile
```
