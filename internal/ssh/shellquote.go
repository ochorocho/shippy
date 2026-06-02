package ssh

import "strings"

// Quote returns s wrapped so it is safe to interpolate as a single argument
// into a POSIX shell command line executed on the remote server.
//
// Remote commands in shippy are built with fmt.Sprintf and run through the
// remote login shell (session.Run / CombinedOutput). Any value that is not a
// trusted literal — file paths derived from local filenames, deploy paths,
// shared entries, extracted database names, etc. — MUST be passed through
// Quote first, otherwise shell metacharacters in that value lead to command
// injection (CWE-78) on the deploy target.
//
// The value is wrapped in single quotes, within which the shell treats every
// character literally. The only character that cannot appear inside single
// quotes is the single quote itself, so each embedded single quote is replaced
// by the standard close-quote, backslash-escaped-quote, reopen-quote sequence
// (see the ReplaceAll below).
func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
