package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/tillkuhn/billy-idle/pkg/tracker"

	_ "modernc.org/sqlite"
)

// useful variables to pass with ldflags during build, for example
// e.g. go run -ldflags="-w -s -X 'main.version=$(shell git describe --tags --abbrev=0)' -X 'main.commit=$(shell git rev-parse --short HEAD)'"
// goreleaser default: '-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser'
// see also https://goreleaser.com/cookbooks/using-main.version/
var (
	version = "latest"
	date    = "now"
	// commit    = ""
	// builtBy   = "go"
)

// main entry point to run sub commands
func main() {
	app := filepath.Base(os.Args[0])
	log.Printf("🎬 %s started version=%s built=%s pid=%d go=%s arch=%s", app, version, date, os.Getpid(), runtime.Version(), runtime.GOARCH)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	ctx, ctxCancel := context.WithCancel(context.Background())

	var opts tracker.Options
	trackCmd := flag.NewFlagSet("track", flag.ExitOnError)
	trackCmd.StringVar(&opts.Cmd, "cmd", "ioreg", "Command to retrieve HIDIdleTime")
	trackCmd.StringVar(&opts.DbDirectory, "db-dir", "./sqlite", "SQLite directory")
	trackCmd.BoolVar(&opts.Debug, "debug", false, "Debug checkpoints")
	trackCmd.BoolVar(&opts.DropCreate, "drop-create", false, "Drop and re-create db schema on startup")
	trackCmd.StringVar(&opts.Env, "env", "default", "Environment")
	trackCmd.DurationVar(&opts.CheckInterval, "interval", 2*time.Second, "Interval to check for idle time")
	trackCmd.DurationVar(&opts.IdleTolerance, "idle", 10*time.Second, "Max tolerated idle time before client enters idle state")
	if len(os.Args) < 2 {
		os.Args = append(os.Args, "help")
	}
	switch os.Args[1] {
	case "track":
		_ = trackCmd.Parse(os.Args[2:])
		if *trackCmd.Bool("h", false, "Show help") {
			trackCmd.PrintDefaults()
			break
		}
		t := tracker.New(&opts)
		go func() {
			t.Track(ctx)
		}()
		sig := <-sigChan
		log.Printf("🛑 Received Signal %v", sig)
		ctxCancel()
		t.WaitClose()
	default:
		fmt.Printf("Usage: %s [command]\n\nAvailable Commands (more coming soon):\n  track    Starts the tracker\n\n", app)
		fmt.Printf("Use \"%s [command] -h\" for more information about a command.\n", app)
	}
}
