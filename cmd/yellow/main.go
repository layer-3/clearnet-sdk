// yellow is the public Clearnet account and transfer CLI.
package main

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/layer-3/clearnet-sdk/internal/yellow"
)

var version = "dev"

func installedVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

func main() {
	// SIGTERM must unwind the same way as Ctrl+C: the interactive shell and the
	// hidden prompts restore the terminal from a deferred handler, and an
	// in-flight journal step must not be skipped.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := yellow.Execute(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr, "", installedVersion())
	stop()
	os.Exit(code)
}
