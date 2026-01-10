package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
)

var rootCmd = &cobra.Command{
	Use:   "shippy",
	Short: "A TYPO3 deployment tool",
	Long:  `Shippy is a minimal, opinionated deployment tool for TYPO3 projects.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", ".shippy.yaml", "config file (default is .shippy.yaml)")
}
