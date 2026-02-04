# azure2aws

A simplified CLI tool for AWS SAML authentication via Azure AD.

`azure2aws` is a streamlined alternative to `saml2aws`, focused exclusively on Azure AD authentication. It provides a cleaner configuration model and simplified workflow for obtaining temporary AWS credentials through SAML.

## Features

- **Azure AD Only**: Dedicated support for Azure AD SAML authentication
- **Unified Profile**: Single `--profile` flag for both configuration and AWS credentials (no confusing dual-profile concept)
- **YAML Configuration**: Clean, readable YAML config at `~/.azure2aws/config.yaml`
- **Secure Password Storage**: Optional system keyring integration
- **MFA Support**: Auto mode using Azure AD default MFA method
- **Standard AWS Credentials**: Saves to standard `~/.aws/credentials` for seamless AWS CLI/SDK integration
- **Exec Mode**: Execute commands with AWS credentials as environment variables
- **Console Access**: Open AWS Console directly with federated login

## Installation

### From Source

```bash
git clone https://github.com/user/azure2aws.git
cd azure2aws
make build
sudo mv bin/azure2aws /usr/local/bin/
```

### Requirements

- Go 1.25 or later (for building from source)
- Azure AD application configured for SAML authentication to AWS

## Quick Start

### 1. Configure a Profile

```bash
azure2aws configure --profile production
```

You'll be prompted for:
- **Azure AD MyApps URL**: The URL to your Azure AD application (e.g., `https://myapps.microsoft.com/signin/AWS/...`)
- **App ID**: Azure AD Application ID (GUID)
- **Username**: Your Azure AD email address
- **AWS Role ARN** (optional): Pre-select an AWS role if you have multiple

### 2. Login and Get Credentials

```bash
azure2aws login --profile production
```

This will:
1. Authenticate with Azure AD (prompting for password and MFA)
2. Retrieve SAML assertion
3. Assume AWS role via STS
4. Save credentials to `~/.aws/credentials`
5. Optionally save password to system keyring

### 3. Use AWS CLI

```bash
aws --profile production sts get-caller-identity
aws --profile production s3 ls
```

## Commands

### `configure`

Configure a profile with Azure AD and AWS settings.

```bash
azure2aws configure --profile <name>
```

**Flags:**
- `--url` - Azure AD MyApps URL (interactive if not provided)
- `--app-id` - Azure AD Application ID (interactive if not provided)
- `--username` - Azure AD username (interactive if not provided)
- `--region` - AWS region (e.g., us-east-1)
- `--output` - AWS CLI output format (json, text, table)
- `--session-duration` - Session duration in seconds (900-43200, default: 3600)

**Example:**
```bash
# Interactive mode
azure2aws configure --profile production

# Non-interactive mode
azure2aws configure --profile production \
  --url "https://myapps.microsoft.com/signin/AWS/xxx" \
  --app-id "xxx-xxx-xxx" \
  --username "user@example.com" \
  --region "us-east-1" \
  --output "json" \
  --session-duration 3600
```

### `login`

Authenticate and retrieve AWS credentials.

```bash
azure2aws login --profile <name>
```

**Flags:**
- `--force` - Force re-authentication even if credentials are valid
- `--skip-prompt` - Skip interactive prompts (use stored credentials)

**Behavior:**
- Checks if credentials already exist and are still valid
- Skips login if credentials won't expire within 15 minutes (use `--force` to override)
- Prompts for password or retrieves from keyring
- Handles Azure AD MFA automatically
- Saves credentials to `~/.aws/credentials`

### `exec`

Execute a command with AWS credentials as environment variables.

```bash
azure2aws exec --profile <name> -- <command> [args...]
```

**Example:**
```bash
azure2aws exec --profile production -- aws s3 ls
azure2aws exec --profile production -- terraform plan
azure2aws exec --profile production -- env | grep AWS
```

**Environment Variables Set:**
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_SESSION_TOKEN`
- `AWS_SECURITY_TOKEN` (legacy)
- `AWS_REGION` / `AWS_DEFAULT_REGION` (if configured)
- `AWS_CREDENTIAL_EXPIRATION`
- `AWS_PROFILE` / `AWS_DEFAULT_PROFILE`

### `console`

Open AWS Management Console in your browser.

```bash
azure2aws console --profile <name>
```

**Flags:**
- `--link` - Print federation URL instead of opening browser
- `--service <name>` - Open specific AWS service (e.g., `ec2`, `s3`)

**Example:**
```bash
azure2aws console --profile production
azure2aws console --profile production --service ec2
azure2aws console --profile production --link  # Print URL only
```

### `version`

Display version information.

```bash
azure2aws version
```

## Configuration

### Config File Structure

Location: `~/.azure2aws/config.yaml`

```yaml
defaults:
  region: us-east-1
  session_duration: 3600

profiles:
  production:
    url: https://myapps.microsoft.com/signin/AWS/xxx-xxx-xxx
    app_id: 12345678-1234-1234-1234-123456789abc
    username: user@example.com
    role_arn: arn:aws:iam::123456789012:role/MyRole  # optional
    region: us-west-2  # optional, overrides default
  
  development:
    url: https://myapps.microsoft.com/signin/AWS/yyy-yyy-yyy
    app_id: 87654321-4321-4321-4321-cba987654321
    username: user@example.com
```

### AWS Credentials File

Location: `~/.aws/credentials`

```ini
[production]
aws_access_key_id = ASIA...
aws_secret_access_key = ...
aws_session_token = ...
region = us-west-2
x_security_token_expires = 2024-02-04T12:00:00Z
```

## Global Flags

- `-p, --profile <name>` - AWS profile name (default: "default")
- `-v, --verbose` - Enable verbose output
- `--debug` - Enable debug mode
- `--config <path>` - Config file path (default: `~/.azure2aws/config.yaml`)

## Security

### Password Storage

Passwords can be stored securely in your system keyring:
- **macOS**: Keychain
- **Linux**: GNOME Keyring, KWallet, or Secret Service
- **Windows**: Windows Credential Manager

When prompted after login, choose "y" to save your password.

To remove stored password:
```bash
# On macOS
security delete-generic-password -s azure2aws -a <profile>

# On Linux (GNOME)
secret-tool clear service azure2aws profile <profile>

# On Windows
cmdkey /delete:azure2aws/<profile>
```

### File Permissions

- Config file: `0600` (read/write owner only)
- Credentials file: `0600` (read/write owner only)
- Config directory: `0700` (rwx owner only)

## Comparison with saml2aws

| Feature | azure2aws | saml2aws |
|---------|-----------|----------|
| **Azure AD Support** | ✅ Dedicated | ✅ One of many |
| **Config Format** | YAML | INI |
| **Profile Concept** | Single unified profile | Dual profile (IDP + AWS) |
| **Config Location** | `~/.azure2aws/config.yaml` | `~/.saml2aws` |
| **MFA Modes** | Auto only (simpler) | Multiple (complex) |
| **Providers** | Azure AD only | 20+ providers |
| **Complexity** | Low | High |

## Troubleshooting

### "credentials have expired"

Run `azure2aws login --profile <name>` to refresh credentials.

### "failed to load credentials"

Ensure you've run `configure` and `login` for the profile:
```bash
azure2aws configure --profile <name>
azure2aws login --profile <name>
```

### "MFA authentication failed"

- Ensure you're using the default MFA method configured in Azure AD
- Check that your MFA device (Authenticator app, SMS, etc.) is accessible
- Try running with `--verbose` to see detailed authentication flow

### "SAML assertion expired"

This usually means the authentication flow took too long. Retry the login command.

## Development

### Building

```bash
make build          # Build for current platform
make test           # Run tests
make lint           # Run linter
make clean          # Clean build artifacts
```

### Project Structure

```
azure2aws/
├── cmd/azure2aws/      # Main entry point
├── internal/
│   ├── cmd/            # CLI commands
│   ├── config/         # Configuration management
│   ├── provider/       # Azure AD authentication
│   ├── aws/            # AWS STS and credentials
│   ├── saml/           # SAML parsing
│   ├── keyring/        # Keyring integration
│   └── prompter/       # Interactive prompts
├── Makefile
└── go.mod
```

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Acknowledgments

Inspired by [saml2aws](https://github.com/Versent/saml2aws) - thanks to the Versent team for the original implementation and Azure AD authentication logic.
