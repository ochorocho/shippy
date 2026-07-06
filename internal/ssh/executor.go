package ssh

import (
	"fmt"
	"os"

	"github.com/ochorocho/shippy/internal/ui"
)

// Command represents a command to execute
type Command struct {
	Name string
	Run  string

	// Context, when non-empty, is a subcontext-entry prefix (e.g.
	// "docker exec php85"). The command — including the `cd` into the release
	// directory — is wrapped as `<Context> sh -c '<cd workDir && Run>'` so it
	// runs inside that context. Empty means run directly on the remote host.
	Context string
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
	// #nosec G104 -- Printf errors in UI output can be safely ignored
	out.Cyan.Printf("→ Executing %d commands on remote server\n", len(commands))
	fmt.Printf("\n")

	for i, cmd := range commands {
		// Print command header
		// #nosec G104 -- Printf errors in UI output can be safely ignored
		out.Cyan.Printf("  [%d/%d] %s\n", i+1, len(commands), cmd.Name)
		// #nosec G104 -- Printf errors in UI output can be safely ignored
		out.Yellow.Printf("  $ %s\n", cmd.Run)
		fmt.Printf("\n")

		fullCmd := buildRemoteCommand(cmd, workDir)

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

// buildRemoteCommand assembles the remote shell command for a single command.
//
// The base form changes into the release directory first (workDir is a path and
// is quoted; cmd.Run is the operator-defined command, passed verbatim):
//
//	cd '<workDir>' && <cmd.Run>
//
// When cmd.Context is set, the whole thing is wrapped so it runs inside that
// subcontext (e.g. a container):
//
//	<cmd.Context> sh -c '<cd ... && cmd.Run>'
//
// The `cd` therefore happens inside the context, so the release path must
// resolve to the same path there — the normal bind-mount case. The context
// prefix is operator-defined and passed verbatim; only the inner command is
// quoted for the `sh -c` argument.
func buildRemoteCommand(cmd Command, workDir string) string {
	inner := cmd.Run
	if workDir != "" {
		inner = fmt.Sprintf("cd %s && %s", Quote(workDir), cmd.Run)
	}

	if cmd.Context != "" {
		return fmt.Sprintf("%s sh -c %s", cmd.Context, Quote(inner))
	}

	return inner
}
