package ui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
)

// Output provides consistent, reusable output formatting
type Output struct {
	Blue   *color.Color
	Green  *color.Color
	Yellow *color.Color
	Red    *color.Color
	Cyan   *color.Color
}

// New creates a new Output with standard colors
func New() *Output {
	return &Output{
		Blue:   color.New(color.FgBlue, color.Bold),
		Green:  color.New(color.FgGreen, color.Bold),
		Yellow: color.New(color.FgYellow),
		Red:    color.New(color.FgRed, color.Bold),
		Cyan:   color.New(color.FgCyan, color.Bold),
	}
}

// Success prints a success message with checkmark
func (o *Output) Success(format string, args ...interface{}) {
	o.Green.Printf("✓ "+format+"\n", args...)
}

// Error prints an error message with X mark
func (o *Output) Error(format string, args ...interface{}) {
	o.Red.Printf("✗ "+format+"\n", args...)
}

// Info prints an informational message
func (o *Output) Info(format string, args ...interface{}) {
	o.Yellow.Printf(format+"\n", args...)
}

// Step prints a step header with arrow
func (o *Output) Step(format string, args ...interface{}) {
	o.Cyan.Printf("→ "+format+"\n", args...)
}

// StepNumber prints a numbered step header
func (o *Output) StepNumber(n int, format string, args ...interface{}) {
	fmt.Println()
	msg := fmt.Sprintf(format, args...)
	o.Blue.Printf("Step %d: %s\n", n, msg)
	fmt.Println()
}

// Header prints a decorative header box
func (o *Output) Header(title string) {
	fmt.Println()
	o.Blue.Println("╔═══════════════════════════════════════════════════════════╗")
	o.Blue.Printf("║  %-57s║\n", title)
	o.Blue.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// HeaderGreen prints a decorative success header box
func (o *Output) HeaderGreen(title string) {
	fmt.Println()
	o.Green.Println("╔═══════════════════════════════════════════════════════════╗")
	o.Green.Printf("║  %-57s║\n", title)
	o.Green.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// ProgressBar formats a progress bar string
func (o *Output) ProgressBar(current, total int, currentItem string, barWidth, maxItemLen int) string {
	// Calculate progress
	percentage := float64(current) / float64(total) * 100
	filled := int(float64(barWidth) * float64(current) / float64(total))

	// Build progress bar
	bar := strings.Repeat("=", filled)
	if filled < barWidth {
		bar += ">"
		bar += strings.Repeat(" ", barWidth-filled-1)
	}

	// Truncate item name if too long
	displayItem := currentItem
	if len(displayItem) > maxItemLen {
		displayItem = "..." + displayItem[len(displayItem)-maxItemLen+3:]
	}

	// Format progress bar
	return fmt.Sprintf("  [%s] %d/%d (%.1f%%) - %s", bar, current, total, percentage, displayItem)
}

// PrintProgressBar prints a progress bar to stdout with carriage return
func (o *Output) PrintProgressBar(current, total int, currentItem string) {
	bar := o.ProgressBar(current, total, currentItem, 40, 50)
	fmt.Print("\r")
	o.Cyan.Print(bar)
	fmt.Print(strings.Repeat(" ", 10)) // Clear remaining chars
}

// ClearLine clears the current line
func (o *Output) ClearLine() {
	fmt.Print("\n")
}

// SelectHost prompts the user to select a host from a list using Bubble Tea
func (o *Output) SelectHost(hosts []string) (string, error) {
	if len(hosts) == 0 {
		return "", fmt.Errorf("no hosts available")
	}

	// If only one host, use it automatically
	if len(hosts) == 1 {
		o.Info("Only one host configured: %s", hosts[0])
		return hosts[0], nil
	}

	// Create and run Bubble Tea selector with input/output options
	model := newSelectorModel(hosts)
	p := tea.NewProgram(
		model,
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stderr),
	)

	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run selector: %w", err)
	}

	m, ok := finalModel.(selectorModel)
	if !ok {
		return "", fmt.Errorf("unexpected model type")
	}

	if m.quitted {
		return "", fmt.Errorf("selection cancelled")
	}

	if m.selected == "" {
		return "", fmt.Errorf("no host selected")
	}

	return m.selected, nil
}
