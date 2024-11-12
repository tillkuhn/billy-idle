package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tillkuhn/billy-idle/pkg/tracker"
)

var opts tracker.Options

// trackCmd represents the track command
var trackCmd = &cobra.Command{
	Use:   "track",
	Short: "Track idle time",
	Long:  `Starts the tracker in daemon mode to record busy and idle times.`,
	Run: func(cmd *cobra.Command, _ []string) {
		dbg, _ := cmd.Flags().GetBool("debug")
		opts.Debug = dbg
		track()
	},
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags which will work for this command
	// Cobra supports local flags which will only run when this command
	rootCmd.AddCommand(trackCmd)
	trackCmd.PersistentFlags().StringVarP(&opts.Env, "env", "e", "default", "Environment")
	trackCmd.PersistentFlags().StringVarP(&opts.AppDir, "app-dir", "a", "", "App Directory e.g. for SQLite DB (defaults to $HOME/.billy-idle/<env>")
	trackCmd.PersistentFlags().StringVarP(&opts.Cmd, "cmd", "c", "ioreg", "Command to retrieve HIDIdleTime")
	trackCmd.PersistentFlags().BoolVar(&opts.DropCreate, "drop-create", false, "Drop and re-create db schema on startup")
	trackCmd.PersistentFlags().DurationVarP(&opts.CheckInterval, "interval", "i", 2*time.Second, "Interval to check for idle time")
	trackCmd.PersistentFlags().DurationVarP(&opts.IdleTolerance, "idle", "m", 10*time.Second, "Max tolerated idle time before client enters idle state")
}

func track() {
	app := filepath.Base(os.Args[0])
	fmt.Printf("🎬 %s started version=%s built=%s pid=%d go=%s arch=%s\n", app, version, date, os.Getpid(), runtime.Version(), runtime.GOARCH)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	ctx, ctxCancel := context.WithCancel(context.Background())
	if opts.AppDir == "" {
		opts.AppDir = defaultAppDir(opts.Env)
	}
	t := tracker.New(&opts)
	// todo: more cases, less ifs .. and ask the cobra for help :-)
	if os.Args[1] == "report" {
		if err := t.Report(ctx, os.Stdout); err != nil {
			log.Println(err)
		}
	}

	go func() {
		t.Track(ctx)
	}()

	sig := <-sigChan
	log.Printf("🔫 Received signal %v, initiate shutdown", sig)
	ctxCancel()
	t.WaitClose()
}

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
