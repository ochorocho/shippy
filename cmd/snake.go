package cmd

import (
	"fmt"
	"os"

	"github.com/ochorocho/shippy/internal/game"
	"github.com/ochorocho/shippy/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// snakeCmd is an easter egg, hidden from the help output.
var snakeCmd = &cobra.Command{
	Use:          "snake",
	Short:        "Play a game of snake",
	Hidden:       true,
	Args:         cobra.NoArgs,
	RunE:         runSnake,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(snakeCmd)
}

func runSnake(cmd *cobra.Command, args []string) error {
	out := ui.New()

	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		err := fmt.Errorf("snake needs an interactive terminal")
		out.Error("%v", err)
		return err
	}

	best, err := game.Play(os.Stdin, os.Stdout)
	if err != nil {
		out.Error("%v", err)
		return err
	}

	out.Success("Thanks for playing! Best score: %d", best)
	return nil
}
