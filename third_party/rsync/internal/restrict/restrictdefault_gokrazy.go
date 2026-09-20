//go:build gokrazy

package restrict

var defaultRoDirs = []string{
	// See restrictdefault_others.go for rationale
	"/etc",
	// On systems with a read-only root file systems (like gokrazy),
	// /etc/resolv.conf is a symlink to /tmp/resolv.conf,
	// so we also need read-only access to /tmp.
	"/tmp",
}
