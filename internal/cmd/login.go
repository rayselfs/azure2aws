package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/azure2aws/internal/aws"
	"github.com/user/azure2aws/internal/config"
	"github.com/user/azure2aws/internal/keyring"
	"github.com/user/azure2aws/internal/prompter"
	"github.com/user/azure2aws/internal/provider"
	"github.com/user/azure2aws/internal/provider/azuread"
	"github.com/user/azure2aws/internal/saml"
)

func newLoginCmd() *cobra.Command {
	var (
		force      bool
		skipPrompt bool
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate and retrieve AWS credentials",
		Long: `Authenticates with Azure AD and retrieves temporary AWS credentials via SAML.

The credentials are stored in ~/.aws/credentials under the specified profile.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(force, skipPrompt)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force re-authentication even if credentials are valid")
	cmd.Flags().BoolVar(&skipPrompt, "skip-prompt", false, "Skip interactive prompts (use stored credentials)")

	return cmd
}

func runLogin(force, skipPrompt bool) error {
	profileName := GetProfile()
	configPath := GetConfigFile()

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w\nRun 'azure2aws configure --profile %s' to set up a profile", err, profileName)
	}

	profile, err := cfg.GetProfile(profileName)
	if err != nil {
		return fmt.Errorf("profile '%s' not found\nRun 'azure2aws configure --profile %s' to set up a profile", profileName, profileName)
	}

	// Check if credentials are still valid (unless force is specified)
	if !force && !aws.CredentialsExpired(profileName) {
		creds, err := aws.LoadCredentials(profileName)
		if err == nil && creds != nil {
			fmt.Printf("Credentials for profile '%s' are still valid (expires: %s)\n", profileName, creds.Expiration.Local().Format("2006-01-02 15:04:05"))
			fmt.Println("Use --force to re-authenticate")
			return nil
		}
	}

	// Get password
	password, err := getPassword(profileName, profile.Username, skipPrompt)
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}

	// Create Azure AD client
	client, err := azuread.NewClient(&azuread.ClientOptions{
		URL:   profile.URL,
		AppID: profile.AppID,
	})
	if err != nil {
		return fmt.Errorf("failed to create Azure AD client: %w", err)
	}

	// Authenticate
	fmt.Printf("Authenticating as %s...\n", profile.Username)
	samlAssertion, err := client.Authenticate(provider.NewLoginCredentials(profile.Username, password))
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Parse SAML assertion to get roles
	roles, err := saml.ParseAssertion(samlAssertion)
	if err != nil {
		return fmt.Errorf("failed to parse SAML assertion: %w", err)
	}

	if len(roles) == 0 {
		return fmt.Errorf("no AWS roles found in SAML assertion")
	}

	// Select role
	var selectedRole *saml.AWSRole
	if len(roles) == 1 {
		selectedRole = roles[0]
		fmt.Printf("Using role: %s\n", selectedRole.Name)
	} else if profile.RoleARN != "" {
		// Use configured role ARN
		for _, role := range roles {
			if role.RoleARN == profile.RoleARN {
				selectedRole = role
				break
			}
		}
		if selectedRole == nil {
			return fmt.Errorf("configured role %s not found in SAML assertion", profile.RoleARN)
		}
	} else {
		// Prompt user to select role
		selectedRole, err = selectRole(roles)
		if err != nil {
			return fmt.Errorf("failed to select role: %w", err)
		}
	}

	samlDuration, _ := saml.ExtractSessionDuration(samlAssertion)
	sessionDuration := aws.GetSessionDuration(profile.SessionDuration, samlDuration)

	fmt.Printf("Assuming role %s...\n", selectedRole.Name)
	creds, err := aws.AssumeRoleWithSAML(selectedRole, samlAssertion, sessionDuration, profile.Region, profile.Output)
	if err != nil {
		return fmt.Errorf("failed to assume role: %w", err)
	}

	if err := aws.SaveCredentials(profileName, creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Printf("\n✓ Credentials saved to profile '%s'\n", profileName)
	fmt.Printf("  Expires: %s\n", creds.Expiration.Local().Format("2006-01-02 15:04:05"))
	if creds.Region != "" {
		fmt.Printf("  Region: %s\n", creds.Region)
	}
	if creds.Output != "" {
		fmt.Printf("  Output: %s\n", creds.Output)
	}

	fmt.Printf("\nTo use this profile, run:\n")
	fmt.Printf("  export AWS_PROFILE=%s\n", profileName)
	fmt.Printf("\nOr use it directly:\n")
	fmt.Printf("  aws --profile %s sts get-caller-identity\n", profileName)

	if !skipPrompt && !keyring.HasPassword(profileName) {
		if savePassword, err := prompter.Confirm("Save password to keyring for future logins?", false); err == nil && savePassword {
			if err := keyring.SavePassword(profileName, password); err != nil {
				fmt.Printf("Warning: Failed to save password: %v\n", err)
			} else {
				fmt.Println("Password saved to keyring.")
			}
		}
	}

	return nil
}

func getPassword(profileName, username string, skipPrompt bool) (string, error) {
	if password, err := keyring.GetPassword(profileName); err == nil && password != "" {
		return password, nil
	}

	// If skip-prompt is set and no password in keyring, fail
	if skipPrompt {
		return "", fmt.Errorf("no password found in keyring and --skip-prompt is set")
	}

	// Prompt for password
	return prompter.Password(fmt.Sprintf("Password for %s", username))
}

// selectRole prompts user to select a role from multiple options
func selectRole(roles []*saml.AWSRole) (*saml.AWSRole, error) {
	if len(roles) == 0 {
		return nil, fmt.Errorf("no roles to select from")
	}

	options := make([]string, len(roles))
	for i, role := range roles {
		options[i] = fmt.Sprintf("%s (Account: %s)", role.Name, role.AccountID())
	}

	idx, err := prompter.Select("Select an AWS role:", options)
	if err != nil {
		return nil, err
	}

	return roles[idx], nil
}
