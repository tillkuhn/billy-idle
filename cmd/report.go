package cmd

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/tillkuhn/billy-idle/pkg/tracker"

	"github.com/spf13/cobra"
)

// reportCmd represents the report command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a report",
	Long:  `Generates a report based on the recorded idle and busy times.`,
	Run: func(_ *cobra.Command, _ []string) {
		run()
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.PersistentFlags().StringVarP(&opts.Env, "env", "e", "default", "Environment")
	reportCmd.PersistentFlags().StringVarP(&opts.AppDir, "app-dir", "a", "", "App Directory e.g. for SQLite DB (defaults to $HOME/.billy-idle/<env>")
	reportCmd.PersistentFlags().DurationVar(&opts.MinBusy, "min-busy", 5*time.Minute, "Minimum time for a busy record to count for the report")
	reportCmd.PersistentFlags().DurationVar(&opts.MaxBusy, "max-busy", 10*time.Hour, "Max allowed time busy period per day (w/o breaks), report only")
	reportCmd.PersistentFlags().DurationVar(&opts.RegBusy, "reg-busy", 7*time.Hour+48*time.Minute, "Regular busy period per day (w/o breaks), report only")
}

func run() {
	// todo: remove redundancy with track cmd
	if opts.AppDir == "" {
		opts.AppDir = defaultAppDir(opts.Env)
	}
	t := tracker.New(&opts)
	if err := t.Report(context.Background(), os.Stdout); err != nil {
		log.Println(err)
	}
}
