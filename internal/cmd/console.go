package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/user/azure2aws/internal/aws"
)

func newConsoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "console",
		Short: "Open AWS Console in browser",
		Long: `Opens the AWS Management Console using federated login.

Uses AWS Federation to create a temporary sign-in URL with your current credentials.

If credentials are expired, an error is returned (use 'azure2aws login' first).

Examples:
  azure2aws console --profile production
  azure2aws console --profile production --link
  azure2aws console --profile production --service ec2`,
		RunE: runConsole,
	}

	cmd.Flags().Bool("link", false, "Print URL instead of opening browser")
	cmd.Flags().String("service", "", "AWS service to open (e.g., ec2, s3)")

	return cmd
}

func runConsole(cmd *cobra.Command, args []string) error {
	profileName := GetProfile()

	creds, err := aws.LoadCredentials(profileName)
	if err != nil {
		return fmt.Errorf("failed to load credentials for profile %q: %w\nRun 'azure2aws login --profile %s' first", profileName, err, profileName)
	}

	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
		return fmt.Errorf("credentials for profile %q are empty\nRun 'azure2aws login --profile %s' first", profileName, profileName)
	}

	if !creds.Expiration.IsZero() && aws.IsExpired(creds.Expiration) {
		return fmt.Errorf("credentials for profile %q have expired at %s\nRun 'azure2aws login --profile %s' to refresh",
			profileName, creds.Expiration.Format(time.RFC3339), profileName)
	}

	service, _ := cmd.Flags().GetString("service")
	loginURL, err := aws.GetFederatedLoginURL(creds, service)
	if err != nil {
		return fmt.Errorf("failed to generate console URL: %w", err)
	}

	linkOnly, _ := cmd.Flags().GetBool("link")
	if linkOnly {
		fmt.Println(loginURL)
		return nil
	}

	if IsVerbose() {
		fmt.Fprintf(os.Stderr, "Opening AWS Console for profile: %s\n", profileName)
	}

	if err := browser.OpenURL(loginURL); err != nil {
		return fmt.Errorf("failed to open browser: %w\nURL: %s", err, loginURL)
	}

	fmt.Println("AWS Console opened in your default browser")
	return nil
}
