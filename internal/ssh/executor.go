package ssh

import (
	"fmt"
	"os"

	"tinnie/internal/ui"
)

// Command represents a command to execute
type Command struct {
	Name string
	Run  string
}

// Executor handles command execution on remote servers
type Executor struct {
	client *Client
}

// NewExecutor creates a new command executor
func NewExecutor(client *Client) *Executor {
	return &Executor{
		client: client,
	}
}

// Execute runs a list of commands sequentially
func (e *Executor) Execute(commands []Command, workDir string) error {
	out := ui.New()

	fmt.Printf("\n")
	out.Cyan.Printf("→ Executing %d commands on remote server\n", len(commands))
	fmt.Printf("\n")

	for i, cmd := range commands {
		// Print command header
		out.Cyan.Printf("  [%d/%d] %s\n", i+1, len(commands), cmd.Name)
		out.Yellow.Printf("  $ %s\n", cmd.Run)
		fmt.Printf("\n")

		// Build full command (cd to work directory first)
		fullCmd := cmd.Run
		if workDir != "" {
			fullCmd = fmt.Sprintf("cd %s && %s", workDir, cmd.Run)
		}

		// Execute command with output streaming
		if err := e.client.RunCommandWithOutput(fullCmd, os.Stdout, os.Stderr); err != nil {
			fmt.Printf("\n")
			out.Error("Command failed: %v", err)
			return fmt.Errorf("command '%s' failed: %w", cmd.Name, err)
		}

		out.Success("Command completed successfully")
		fmt.Printf("\n")
	}

	out.Success("All commands executed successfully")

	return nil
}
