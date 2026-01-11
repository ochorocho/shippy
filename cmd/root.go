package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
)

var rootCmd = &cobra.Command{
	Use:          "shippy",
	Short:        "A TYPO3 deployment tool",
	Long:         `Shippy is a minimal, opinionated deployment tool for TYPO3 projects.`,
	SilenceErrors: true, // Commands handle their own error display
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
}
