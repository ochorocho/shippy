package ui

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/blacktop/go-termimg"
)

//go:embed logo.png
var logoPNG []byte

const (
	logoW    = 48 // termimg width parameter
	logoH    = 24 // termimg height parameter
	logoCols = 24 // approximate character cells the image occupies
	logoRows = 12 // approximate terminal rows the image occupies
)

// PrintLogo prints the embedded PNG logo with title text beside it
func (o *Output) PrintLogo() {
	fmt.Println()
	img, err := termimg.From(bytes.NewReader(logoPNG))
	if err != nil {
		return
	}
	if err := img.Width(logoW).Height(logoH).Print(); err != nil {
		return
	}

	// Use ANSI cursor movement to place text to the right of the image
	textCol := logoCols + 3 // image width + gap
	titleRow := logoRows/2 - 1

	// Move cursor up from bottom of image to title position
	fmt.Printf("\033[%dA", logoRows-titleRow)
	// Move right past the image and print bold orange title
	fmt.Printf("\033[%dC\033[1;38;2;255;135;0mShippy\033[0m\n", textCol)
	// Move right again for subtitle
	fmt.Printf("\033[%dCMinimal, opinionated deployment\n", textCol)
	fmt.Printf("\033[%dCtool for PHP projects.\n", textCol)
	// Move cursor back down to clear the image area
	remaining := logoRows - titleRow - 2
	if remaining > 0 {
		fmt.Printf("\033[%dB", remaining)
	}
	fmt.Println()
}
