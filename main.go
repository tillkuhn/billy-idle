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

// Useful variables passed with ldflags during build, see goreleaser https://goreleaser.com/cookbooks/using-main.version/
var (
	version = "latest"
	date    = "now"
	// unused: commit, builtBy
)

// main entry point to run subcommands
func main() {
	app := filepath.Base(os.Args[0])
	fmt.Printf("🎬 %s started version=%s built=%s pid=%d go=%s arch=%s\n", app, version, date, os.Getpid(), runtime.Version(), runtime.GOARCH)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	ctx, ctxCancel := context.WithCancel(context.Background())

	var opts tracker.Options
	trackCmd := flag.NewFlagSet("track", flag.ExitOnError)
	trackCmd.StringVar(&opts.Cmd, "cmd", "ioreg", "Command to retrieve HIDIdleTime")
	trackCmd.StringVar(&opts.AppDir, "app-dir", "", "App Directory e.g. for SQLite DB (defaults to $HOME/.billy-idle/<env>")
	trackCmd.BoolVar(&opts.Debug, "debug", false, "Debug checkpoints")
	trackCmd.BoolVar(&opts.DropCreate, "drop-create", false, "Drop and re-create db schema on startup")
	trackCmd.StringVar(&opts.Env, "env", "default", "Environment")
	trackCmd.DurationVar(&opts.CheckInterval, "interval", 2*time.Second, "Interval to check for idle time")
	trackCmd.DurationVar(&opts.IdleTolerance, "idle", 10*time.Second, "Max tolerated idle time before client enters idle state")
	trackCmd.DurationVar(&opts.MinBusy, "min-busy", 5*time.Minute, "Minimum time for a busy record to count for the report")
	if len(os.Args) < 2 {
		os.Args = append(os.Args, "help")
	}

	switch os.Args[1] {
	case "track", "report":
		if err := trackCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal(err) // -h and -help will print usage implicitly
		}
		if opts.AppDir == "" {
			opts.AppDir = defaultAppDir(opts.Env)
		}
		t := tracker.New(&opts)
		// todo: more cases, less ifs .. and ask the cobra for help :-)
		if os.Args[1] == "report" {
			if err := t.Report(ctx, os.Stdout); err != nil {
				log.Println(err)
			}
			break
		}

		go func() {
			t.Track(ctx)
		}()

		sig := <-sigChan
		log.Printf("🔫 Received signal %v, initiate shutdown", sig)
		ctxCancel()
		t.WaitClose()
	default:
		fmt.Printf("Usage: %s [command]\n\nAvailable Commands (more coming soon):\n  track    Starts the tracker\n\n", app)
		fmt.Printf("Use \"%s [command] -h\" for more information about a command.\n", app)
	}
}

// defaultAppDir returns the default applications directory in $HOME, e.g. $HOME/.billy-idle/<env>
func defaultAppDir(env string) string {
	home, err := os.UserHomeDir() // $HOME on *nix
	if err != nil {
		log.Fatal(err)
	}
	dir := filepath.Join(home, ".billy-idle", env)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		log.Fatal(err)
	}
	return dir
}
