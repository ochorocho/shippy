package ssh

import (
	"testing"
	"time"
)

func TestParseTimeout(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "integer seconds",
			input:    "30",
			expected: 30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "seconds with unit",
			input:    "45s",
			expected: 45 * time.Second,
			wantErr:  false,
		},
		{
			name:     "minutes",
			input:    "5m",
			expected: 5 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "hours",
			input:    "2h",
			expected: 2 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "milliseconds",
			input:    "500ms",
			expected: 500 * time.Millisecond,
			wantErr:  false,
		},
		{
			name:     "combined duration",
			input:    "1h30m",
			expected: 90 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "zero seconds",
			input:    "0",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "large value",
			input:    "3600",
			expected: 3600 * time.Second,
			wantErr:  false,
		},
		{
			name:     "invalid format - letters only",
			input:    "invalid",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "negative value (parsed as -10 seconds)",
			input:    "-10",
			expected: -10 * time.Second,
			wantErr:  false,
		},
		{
			name:     "invalid format - empty string",
			input:    "",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "invalid format - special characters",
			input:    "@#$",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTimeout(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimeout(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("parseTimeout(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseTimeoutEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "whitespace trimming not supported",
			input:    " 30 ",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "float as string",
			input:    "30.5",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "very large number",
			input:    "999999",
			expected: 999999 * time.Second,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTimeout(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimeout(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("parseTimeout(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
