package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/azure2aws/internal/config"
	"github.com/user/azure2aws/internal/keyring"
	"github.com/user/azure2aws/internal/prompter"
)

func newConfigureCmd() *cobra.Command {
	var (
		flagURL             string
		flagAppID           string
		flagUsername        string
		flagRegion          string
		flagOutput          string
		flagSessionDuration int
	)

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure a profile",
		Long: `Interactively configure an Azure AD SAML profile.

This will prompt for:
- Azure AD app URL
- Azure AD application ID  
- Username/email
- AWS region (optional)
- AWS CLI output format (optional)
- Session duration (optional)

If --url, --app-id, and --username flags are all provided,
the command runs in non-interactive mode.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigure(flagURL, flagAppID, flagUsername, flagRegion, flagOutput, flagSessionDuration)
		},
	}

	cmd.Flags().StringVar(&flagURL, "url", "", "Azure AD app URL (non-interactive)")
	cmd.Flags().StringVar(&flagAppID, "app-id", "", "Azure AD application ID (non-interactive)")
	cmd.Flags().StringVar(&flagUsername, "username", "", "Username/email (non-interactive)")
	cmd.Flags().StringVar(&flagRegion, "region", "", "AWS region (e.g., us-east-1)")
	cmd.Flags().StringVar(&flagOutput, "output", "", "AWS CLI output format (json, text, table)")
	cmd.Flags().IntVar(&flagSessionDuration, "session-duration", 0, "Session duration in seconds (900-43200, default: 3600)")

	return cmd
}

func runConfigure(flagURL, flagAppID, flagUsername, flagRegion, flagOutput string, flagSessionDuration int) error {
	profileName := GetProfile()
	configPath := GetConfigFile()

	cfg, err := config.LoadOrCreateConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var existingProfile config.Profile
	if cfg.HasProfile(profileName) {
		mp, _ := cfg.GetProfile(profileName)
		existingProfile = config.Profile{
			URL:             mp.URL,
			AppID:           mp.AppID,
			Username:        mp.Username,
			RoleARN:         mp.RoleARN,
			Region:          mp.Region,
			Output:          mp.Output,
			SessionDuration: mp.SessionDuration,
		}
		fmt.Printf("Updating existing profile: %s\n", profileName)
	} else {
		fmt.Printf("Creating new profile: %s\n", profileName)
	}

	nonInteractive := flagURL != "" && flagAppID != "" && flagUsername != ""

	var newProfile config.Profile

	if nonInteractive {
		newProfile = config.Profile{
			URL:             flagURL,
			AppID:           flagAppID,
			Username:        flagUsername,
			Region:          flagRegion,
			Output:          flagOutput,
			SessionDuration: flagSessionDuration,
		}
	} else {
		p := prompter.New()

		defaultURL := existingProfile.URL
		if flagURL != "" {
			defaultURL = flagURL
		}
		url, err := p.PromptString("Azure AD App URL", defaultURL)
		if err != nil {
			return err
		}

		defaultAppID := existingProfile.AppID
		if flagAppID != "" {
			defaultAppID = flagAppID
		}
		appID, err := p.PromptString("Azure AD Application ID", defaultAppID)
		if err != nil {
			return err
		}

		defaultUsername := existingProfile.Username
		if flagUsername != "" {
			defaultUsername = flagUsername
		}
		username, err := p.PromptString("Username (email)", defaultUsername)
		if err != nil {
			return err
		}

		defaultRegion := existingProfile.Region
		if flagRegion != "" {
			defaultRegion = flagRegion
		}
		if defaultRegion == "" {
			defaultRegion = cfg.Defaults.Region
		}
		region, err := p.PromptString("AWS Region", defaultRegion)
		if err != nil {
			return err
		}

		defaultOutput := existingProfile.Output
		if flagOutput != "" {
			defaultOutput = flagOutput
		}
		if defaultOutput == "" {
			defaultOutput = "json"
		}
		output, err := p.PromptString("AWS CLI output format (json/text/table)", defaultOutput)
		if err != nil {
			return err
		}

		defaultSessionDuration := existingProfile.SessionDuration
		if flagSessionDuration > 0 {
			defaultSessionDuration = flagSessionDuration
		}
		if defaultSessionDuration == 0 {
			defaultSessionDuration = cfg.Defaults.SessionDuration
		}
		sessionDurationStr := fmt.Sprintf("%d", defaultSessionDuration)
		sessionDurationInput, err := p.PromptString("Session duration in seconds (900-43200)", sessionDurationStr)
		if err != nil {
			return err
		}
		var sessionDuration int
		if sessionDurationInput != "" {
			if _, err := fmt.Sscanf(sessionDurationInput, "%d", &sessionDuration); err != nil {
				return fmt.Errorf("invalid session duration: %w", err)
			}
			if sessionDuration < 900 || sessionDuration > 43200 {
				return fmt.Errorf("session duration must be between 900 and 43200 seconds")
			}
		} else {
			sessionDuration = defaultSessionDuration
		}

		newProfile = config.Profile{
			URL:             url,
			AppID:           appID,
			Username:        username,
			Region:          region,
			Output:          output,
			SessionDuration: sessionDuration,
		}

		if keyring.IsAvailable() {
			savePassword, err := p.PromptConfirm("Save password to keyring?", false)
			if err != nil {
				return err
			}

			if savePassword {
				password, err := p.PromptPassword("Password")
				if err != nil {
					return err
				}

				if password != "" {
					if err := keyring.SavePassword(profileName, password); err != nil {
						fmt.Printf("Warning: Failed to save password to keyring: %v\n", err)
					} else {
						fmt.Println("Password saved to keyring.")
					}
				}
			}
		}
	}

	if newProfile.URL == "" {
		return fmt.Errorf("URL is required")
	}
	if newProfile.AppID == "" {
		return fmt.Errorf("App ID is required")
	}
	if newProfile.Username == "" {
		return fmt.Errorf("Username is required")
	}
	if newProfile.SessionDuration > 0 {
		if newProfile.SessionDuration < 900 || newProfile.SessionDuration > 43200 {
			return fmt.Errorf("session duration must be between 900 and 43200 seconds")
		}
	}

	cfg.SetProfile(profileName, newProfile)

	if err := config.SaveConfig(cfg, configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("\nProfile '%s' saved to %s\n", profileName, configPath)
	fmt.Println("\nConfiguration:")
	fmt.Printf("  URL:      %s\n", newProfile.URL)
	fmt.Printf("  App ID:   %s\n", newProfile.AppID)
	fmt.Printf("  Username: %s\n", newProfile.Username)
	if newProfile.Region != "" {
		fmt.Printf("  Region:   %s\n", newProfile.Region)
	}
	if newProfile.Output != "" {
		fmt.Printf("  Output:   %s\n", newProfile.Output)
	}
	if newProfile.SessionDuration > 0 {
		fmt.Printf("  Session Duration: %d seconds (%d hours)\n", newProfile.SessionDuration, newProfile.SessionDuration/3600)
	}

	return nil
}
