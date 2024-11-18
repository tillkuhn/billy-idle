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

// trackOpts represents options to configure the tracker instance
var trackOpts tracker.Options

// trackCmd represents the track command
var trackCmd = &cobra.Command{
	Use:   "track",
	Short: "Track idle time in the background",
	Long:  `Starts the tracker in daemon mode to record busy and idle times.`,
	Run: func(cmd *cobra.Command, _ []string) {
		dbg, _ := cmd.Flags().GetBool("debug")
		trackOpts.Debug = dbg
		track(cmd.Context())
	},
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags which will work for this command
	// Cobra supports local flags which will only run when this command
	rootCmd.AddCommand(trackCmd)
	trackCmd.PersistentFlags().StringVarP(&trackOpts.Env, "env", "e", defaultEnv(), "Environment")
	trackCmd.PersistentFlags().StringVarP(&trackOpts.AppRoot, "app-root", "a", defaultAppRoot(), "App root Directory e.g. for SQLite DB (defaults to $HOME/.billy-idle")
	trackCmd.PersistentFlags().StringVarP(&trackOpts.Cmd, "cmd", "c", "ioreg", "Command to retrieve HIDIdleTime")
	trackCmd.PersistentFlags().BoolVar(&trackOpts.DropCreate, "drop-create", false, "Drop and re-create db schema on startup")
	trackCmd.PersistentFlags().DurationVarP(&trackOpts.CheckInterval, "interval", "i", 2*time.Second, "Interval to check for idle time")
	trackCmd.PersistentFlags().DurationVarP(&trackOpts.IdleTolerance, "idle", "m", 10*time.Second, "Max tolerated idle time before client enters idle state")
}

func track(ctx context.Context) {
	app := filepath.Base(os.Args[0])
	fmt.Printf("🎬 %s started version=%s built=%s pid=%d go=%s arch=%s\n", app, version, date, os.Getpid(), runtime.Version(), runtime.GOARCH)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	t := tracker.New(&trackOpts)
	go func() {
		t.Track(ctx)
	}()

	sig := <-sigChan
	log.Printf("🔫 Received signal %v, initiate shutdown", sig)
	ctxCancel()
	t.WaitClose()
}
