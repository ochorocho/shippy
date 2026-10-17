package ui

import (
	"testing"

	"github.com/blacktop/go-termimg"
)

// The text beside the logo is positioned for a logoCols x logoRows cell area,
// so every protocol must render the logo into exactly that area.
func TestLogoSizeFitsTextLayoutForEveryProtocol(t *testing.T) {
	tests := []struct {
		protocol termimg.Protocol
		wantW    int
		wantH    int
	}{
		// Graphics protocols take width/height in character cells.
		{termimg.Kitty, logoCols, logoRows},
		{termimg.ITerm2, logoCols, logoRows},
		{termimg.Sixel, logoCols, logoRows},
		// Halfblocks (mosaic) packs 2x2 image pixels into one cell.
		{termimg.Halfblocks, logoCols * 2, logoRows * 2},
	}

	for _, tt := range tests {
		t.Run(tt.protocol.String(), func(t *testing.T) {
			w, h := logoSize(tt.protocol)
			if w != tt.wantW || h != tt.wantH {
				t.Errorf("logoSize(%s) = %dx%d, want %dx%d", tt.protocol, w, h, tt.wantW, tt.wantH)
			}
		})
	}
}
