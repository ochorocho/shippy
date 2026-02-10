package cmd

import (
	"fmt"
	"os"
	"strings"

	"shippy/internal/ui"

	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	showVersion bool
)

var rootCmd = &cobra.Command{
	Use:           "shippy",
	Short:         "A TYPO3 deployment tool",
	Long:          `Shippy is a minimal, opinionated deployment tool for Composer based PHP projects.`,
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
		// Cobra's unknown command errors are silenced (SilenceErrors=true).
		// Print a user-friendly message instead.
		errMsg := err.Error()
		if _, after, ok := strings.Cut(errMsg, `unknown command "`); ok {
			if cmdName, _, ok := strings.Cut(after, `"`); ok {
				out := ui.New()
				out.Error("The command %s does not exist!", cmdName)
			}
		}
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", ".shippy.yaml", "config file (default is .shippy.yaml)")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "print version information")

	// Custom help function that shows the logo before help text
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd == rootCmd {
			out := ui.New()
			out.PrintLogo()
		}
		defaultHelp(cmd, args)
	})
}

// printVersion outputs the version information
func printVersion() {
	fmt.Printf("shippy version %s\n", Version)
}
