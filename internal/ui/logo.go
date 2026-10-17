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
	logoCols = 24 // character cells the image occupies
	logoRows = 12 // terminal rows the image occupies
)

// logoSize returns the termimg Width/Height for the given protocol so the
// logo always covers logoCols x logoRows cells. Graphics protocols (Kitty,
// iTerm2, Sixel) take cells; Halfblocks takes image pixels, 2x2 per cell.
func logoSize(p termimg.Protocol) (w, h int) {
	if p == termimg.Halfblocks {
		return logoCols * 2, logoRows * 2
	}
	return logoCols, logoRows
}

// PrintLogo prints the embedded PNG logo with title text beside it
func (o *Output) PrintLogo() {
	fmt.Println()
	img, err := termimg.From(bytes.NewReader(logoPNG))
	if err != nil {
		return
	}
	protocol := termimg.DetectProtocol()
	w, h := logoSize(protocol)
	if err := img.Protocol(protocol).Width(w).Height(h).Print(); err != nil {
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
	fmt.Printf("\033[%dCtool for Composer based PHP projects.\n", textCol)
	// Move cursor back down to clear the image area
	remaining := logoRows - titleRow - 2
	if remaining > 0 {
		fmt.Printf("\033[%dB", remaining)
	}
	fmt.Println()
}
