package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/user/azure2aws/internal/logging"
)

var (
	cfgFile string
	profile string
	verbose bool
	debug   bool
)

// NewRootCmd creates the root command
func NewRootCmd(version, commit, date string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "azure2aws",
		Short: "AWS credentials via Azure AD SAML authentication",
		Long: `azure2aws is a CLI tool that authenticates via Azure AD and 
retrieves temporary AWS credentials using SAML.

Simplified alternative to saml2aws, focused on Azure AD only.`,
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logging.InitLogger(verbose, debug)

			if cfgFile == "" {
				home, err := os.UserHomeDir()
				if err == nil {
					cfgFile = filepath.Join(home, ".azure2aws", "config.yaml")
				}
			}
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "default", "AWS profile name")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug mode")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default: ~/.azure2aws/config.yaml)")

	// Add subcommands
	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newConfigureCmd())
	rootCmd.AddCommand(newExecCmd())
	rootCmd.AddCommand(newConsoleCmd())
	rootCmd.AddCommand(newVersionCmd(version, commit, date))
	rootCmd.AddCommand(newUpdateCmd(version))

	return rootCmd
}

// GetProfile returns the current profile name
func GetProfile() string {
	return profile
}

// GetConfigFile returns the config file path
func GetConfigFile() string {
	return cfgFile
}

// IsVerbose returns whether verbose mode is enabled
func IsVerbose() bool {
	return verbose
}

// IsDebug returns whether debug mode is enabled
func IsDebug() bool {
	return debug
}
