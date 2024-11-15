package cmd

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Useful variables passed with ldflags during build, see goreleaser https://goreleaser.com/cookbooks/using-main.version/
var (
	version = "latest"
	date    = "now"
	// unused: commit, builtBy
)

var ctxCancel context.CancelFunc

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "billy-idle",
	Short: "Simple busy / idle time tracker inspired by the ancient article 'Inactivity and Idle Time on OS X'.",
	Long: `Simple busy / idle time tracker based on the macOS timer called HIDIdleTime that tracks the last time you interacted with the computer, e.g. moved the mouse, typed a key, or interacted with the computer.

billy-idle simply queries this value periodically using the ioreg utility that ships with macOS, and matches it against a pre-defined threshold. 
If exceeded, it will create a record for the busy time period in database. This data can later be used as input for time tracking tools or statistics.`,
	// Uncomment the following line if your bare application has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	var ctx context.Context
	ctx, ctxCancel = context.WithCancel(context.Background())
	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&trackOpts.Debug, "debug", "d", false, "Debug checkpoints")
}

// defaultAppRoot returns the default app root directory
func defaultAppRoot() string {
	home, err := os.UserHomeDir() // $HOME on *nix
	if err != nil {
		log.Fatal(err)
	}
	return home + string(os.PathSeparator) + ".billy-idle"
}

// defaultEnv returns the default environment directory depending on whether we're running in test mode or with go run
// example arg for go run ...: /var/folders/9w/4543534/T/go-build1898714561/b001/exe/main
func defaultEnv() string {
	if testing.Testing() { // https://stackoverflow.com/a/78532310/4292075
		return "test"
	} else if strings.HasSuffix(os.Args[0], "/exe/main") {
		return "dev"
	}
	return "default"
}
