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

// Println prints a line with a newline
func (o *Output) Println(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// EmptyLine prints an empty line
func (o *Output) EmptyLine() {
	fmt.Println()
}

// PrintYAML prints YAML content with syntax highlighting
func (o *Output) PrintYAML(yamlContent string) {
	lines := strings.Split(yamlContent, "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			fmt.Println()
			continue
		}

		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			fmt.Printf("\033[90m%s\033[0m\n", line)
			continue
		}

		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			key := parts[0]
			value := ""
			if len(parts) > 1 {
				value = parts[1]
			}

			indent := len(key) - len(strings.TrimLeft(key, " "))
			keyTrimmed := strings.TrimSpace(key)
			fmt.Printf("%s\033[36m%s\033[0m:", strings.Repeat(" ", indent), keyTrimmed)

			if value != "" {
				valueTrimmed := strings.TrimSpace(value)
				if valueTrimmed == "true" || valueTrimmed == "false" {
					fmt.Printf(" \033[33m%s\033[0m\n", valueTrimmed)
				} else if isNumber(valueTrimmed) {
					fmt.Printf(" \033[35m%s\033[0m\n", valueTrimmed)
				} else {
					fmt.Printf(" \033[32m%s\033[0m\n", valueTrimmed)
				}
			} else {
				fmt.Println()
			}
			continue
		}

		if strings.Contains(line, "- ") {
			parts := strings.SplitN(line, "- ", 2)
			indent := parts[0]
			value := ""
			if len(parts) > 1 {
				value = parts[1]
			}
			fmt.Printf("%s\033[90m-\033[0m \033[32m%s\033[0m\n", indent, value)
			continue
		}

		fmt.Println(line)
	}
}

// SelectHost prompts the user to select a host from a list using Bubble Tea
func (o *Output) SelectHost(hosts []string) (string, error) {
	if len(hosts) == 0 {
		return "", fmt.Errorf("no hosts available")
	}

	if len(hosts) == 1 {
		o.Info("Only one host configured: %s", hosts[0])
		return hosts[0], nil
	}

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

func isNumber(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
