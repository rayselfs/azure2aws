package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/azure2aws/internal/aws"
)

func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [flags] -- command [args...]",
		Short: "Execute a command with AWS credentials",
		Long: `Executes a command with AWS credentials set as environment variables.

The following environment variables are set:
- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
- AWS_SESSION_TOKEN
- AWS_REGION
- AWS_DEFAULT_REGION
- AWS_CREDENTIAL_EXPIRATION

If credentials are expired, an error is returned (use 'azure2aws login' first).

Example:
  azure2aws exec --profile production -- aws s3 ls
  azure2aws exec --profile production -- env | grep AWS`,
		RunE:               runExec,
		DisableFlagParsing: false,
	}

	return cmd
}

func runExec(cmd *cobra.Command, args []string) error {
	cmdArgs := args
	for i, arg := range os.Args {
		if arg == "--" {
			cmdArgs = os.Args[i+1:]
			break
		}
	}

	if len(cmdArgs) == 0 {
		return fmt.Errorf("command to execute is required\n\nUsage: azure2aws exec [flags] -- command [args...]")
	}

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

	if IsVerbose() {
		fmt.Fprintf(os.Stderr, "Using credentials for profile: %s\n", profileName)
		if !creds.Expiration.IsZero() {
			fmt.Fprintf(os.Stderr, "Credentials expire at: %s\n", creds.Expiration.Format(time.RFC3339))
		}
	}

	envVars := buildEnvVars(creds, profileName)
	return execCommand(cmdArgs, envVars)
}

func buildEnvVars(creds *aws.Credentials, profile string) []string {
	vars := []string{
		fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", creds.AccessKeyID),
		fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", creds.SecretAccessKey),
		fmt.Sprintf("AWS_SESSION_TOKEN=%s", creds.SessionToken),
		fmt.Sprintf("AWS_SECURITY_TOKEN=%s", creds.SessionToken),
	}

	if creds.Region != "" {
		vars = append(vars,
			fmt.Sprintf("AWS_REGION=%s", creds.Region),
			fmt.Sprintf("AWS_DEFAULT_REGION=%s", creds.Region),
		)
	}

	if !creds.Expiration.IsZero() {
		vars = append(vars, fmt.Sprintf("AWS_CREDENTIAL_EXPIRATION=%s", creds.Expiration.Format(time.RFC3339)))
	}

	vars = append(vars,
		fmt.Sprintf("AWS_PROFILE=%s", profile),
		fmt.Sprintf("AWS_DEFAULT_PROFILE=%s", profile),
	)

	return vars
}

func execCommand(cmdline []string, envVars []string) error {
	execCmd := exec.Command(cmdline[0], cmdline[1:]...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Env = append(os.Environ(), envVars...)

	err := execCmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}
