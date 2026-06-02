package ssh

import (
	"os/exec"
	"testing"
)

func TestQuote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple path", "/var/www/app", "'/var/www/app'"},
		{"empty string", "", "''"},
		{"with space", "/var/www/my app", "'/var/www/my app'"},
		{"single quote", "it's", `'it'\''s'`},
		{"command substitution", "$(rm -rf /)", "'$(rm -rf /)'"},
		{"backticks", "`id`", "'`id`'"},
		{"semicolon chain", "a; rm -rf ~", "'a; rm -rf ~'"},
		{"pipe", "a | sh", "'a | sh'"},
		{"dollar var", "$HOME", "'$HOME'"},
		{"newline", "a\nrm -rf /", "'a\nrm -rf /'"},
		{"only single quote", "'", `''\'''`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Quote(tt.input); got != tt.want {
				t.Errorf("Quote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestQuoteRoundTripsThroughShell is the real anti-injection guarantee: when
// Quote(s) is substituted into a command and parsed by /bin/sh, the shell must
// reconstruct exactly s as a single argument — no command substitution, no word
// splitting, no glob expansion, no breaking out of the quotes.
func TestQuoteRoundTripsThroughShell(t *testing.T) {
	inputs := []string{
		"normal",
		"/var/www/app",
		"/var/www/my app",
		"$(touch /tmp/pwned)",
		"foo`whoami`bar",
		"x'; touch /tmp/pwned; echo '",
		"a && b || c",
		"* ? [a-z]",
		"$HOME ${PATH}",
		"tab\tand spaces",
		"new\nline",
		"trailing-quote'",
		"only'quote",
	}

	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			// printf %s receives the shell-parsed argument and echoes it back
			// verbatim, so stdout must equal the original input exactly.
			// #nosec G204 -- this test deliberately feeds Quote() output to a shell
			out, err := exec.Command("/bin/sh", "-c", "printf %s "+Quote(in)).Output()
			if err != nil {
				t.Fatalf("shell execution failed for %q: %v", in, err)
			}
			if string(out) != in {
				t.Errorf("round-trip mismatch:\n  input    = %q\n  Quote()  = %q\n  shell out= %q", in, Quote(in), string(out))
			}
		})
	}
}
