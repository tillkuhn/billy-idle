package main

import (
	"context"
	"flag"
	"github.com/tillkuhn/billy-idle/internal/version"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/tillkuhn/billy-idle/pkg/tracker"

	_ "modernc.org/sqlite"
)

// main runs the tracker
func main() {
	log.Printf("🎬 %s started version=%s commit=%s pid=%d",
		filepath.Base(os.Args[0]), version.Version, version.GitCommit, os.Getpid())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	ctx, ctxCancel := context.WithCancel(context.Background())

	var opts tracker.Options
	flag.StringVar(&opts.Cmd, "cmd", "ioreg", "Command to retrieve HIDIdleTime")
	flag.StringVar(&opts.DbDirectory, "db-dir", "./sqlite", "SQLite directory")
	flag.BoolVar(&opts.DropCreate, "drop-create", false, "Drop and re-create db schema on startup")
	flag.StringVar(&opts.Env, "env", "default", "Environment")
	flag.DurationVar(&opts.CheckInterval, "interval", 2*time.Second, "Interval to check for idle time")
	flag.DurationVar(&opts.IdleAfter, "idle", 10*time.Second, "Max time before client is considered idle")
	if len(os.Args) > 1 && os.Args[1] == "help" {
		flag.PrintDefaults()
		return
	}
	flag.Parse()

	t := tracker.New(&opts)
	go func() {
		t.Track(ctx)
	}()

	sig := <-sigChan
	log.Printf("🛑 Received Signal %v", sig)
	ctxCancel()
	t.WaitClose()
}
