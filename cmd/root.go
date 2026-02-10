package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	showVersion bool
)

var rootCmd = &cobra.Command{
	Use:           "shippy",
	Short:         "A TYPO3 deployment tool",
	Long:          `Shippy is a minimal, opinionated deployment tool for TYPO3 projects.`,
	SilenceErrors: true, // Commands handle their own error display
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			runVersion(cmd, args)
			return
		}
		// #nosec G104 -- Help() errors can be safely ignored
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Error already printed by command's out.Error()
		// Just exit with error code
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", ".shippy.yaml", "config file (default is .shippy.yaml)")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "print version information")
}

// printVersion outputs the version information
func printVersion() {
	fmt.Printf("shippy version %s\n", Version)
}
