package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"shippy/internal/ui"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Print all environment variables",
	Long: `Print all environment variables available to Shippy.

This is useful for debugging configuration that uses environment variable
substitution with the ${ENV_VAR} syntax.

Example:
  shippy env
  shippy env | grep DEPLOY`,
	RunE: runEnv,
}

func init() {
	rootCmd.AddCommand(envCmd)
}

func runEnv(cmd *cobra.Command, args []string) error {
	out := ui.New()

	// Get all environment variables
	environ := os.Environ()
	sort.Strings(environ)

	fmt.Printf("\n")
	// #nosec G104 -- Printf errors in UI output can be safely ignored
	out.Cyan.Println("Environment Variables")
	// #nosec G104 -- Printf errors in UI output can be safely ignored
	out.Cyan.Println("=====================")
	fmt.Printf("\n")

	for _, env := range environ {
		// #nosec G104 -- Printf errors in UI output can be safely ignored
		out.Yellow.Println(env)
	}

	fmt.Printf("\n")
	// #nosec G104 -- Printf errors in UI output can be safely ignored
	out.Cyan.Printf("Total: %d variables\n", len(environ))
	fmt.Printf("\n")

	return nil
}
