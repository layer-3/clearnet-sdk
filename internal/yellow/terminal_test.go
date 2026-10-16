package yellow

import (
	"context"
	"os"
	"os/signal"
	"testing"
)

// Driven under a pseudo-terminal by scripts/test-yellow-terminal.py. Default
// Go tests never need a real terminal, a gateway or a private repository.
func TestTerminalDriver(t *testing.T) {
	dir := os.Getenv("YELLOW_TEST_TERMINAL_DIR")
	if dir == "" {
		t.Skip("terminal acceptance driver")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	args := []string{"--json", "account", "import", "terminal-test"}
	if os.Getenv("YELLOW_TEST_TERMINAL_MODE") == "shell" {
		args = []string{"shell"}
		if os.Getenv("YELLOW_TEST_NO_COLOR") == "1" {
			args = append(args, "--no-color")
		}
	}
	if os.Getenv("YELLOW_TEST_TERMINAL_MODE") == "export" {
		args = []string{"account", "export-key", "terminal-test"}
	}
	code := Execute(ctx, args, os.Stdin, os.Stdout, os.Stderr, dir, "test")
	stop()
	os.Exit(code)
}
