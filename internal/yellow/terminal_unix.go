//go:build !windows

package yellow

func terminalDevice() string { return "/dev/tty" }
