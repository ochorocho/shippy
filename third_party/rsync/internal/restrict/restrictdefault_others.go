//go:build !gokrazy

package restrict

var defaultRoDirs = []string{
	// rsync needs /etc/passwd and /etc/group for user/group lookup.
	//
	// As of Go 1.24, the net package Go resolver reads
	// the following DNS configurations files:
	//
	// - /etc/resolv.conf
	// - /etc/hosts
	// - /etc/services
	// - /etc/nsswitch.conf
	//
	// Because the /etc/resolv.conf file might be re-created (by DHCP
	// clients, Tailscale, or similar), we need to provide the entire
	// /etc directory instead of individual files. Otherwise, the
	// program seems to work at first and then fails DNS resolution
	// after a while.
	"/etc",
}
