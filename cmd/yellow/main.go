// yellow is the public Clearnet account and transfer CLI.
package main

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"

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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := yellow.Execute(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr, "", installedVersion())
	stop()
	os.Exit(code)
}
