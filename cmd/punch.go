package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tillkuhn/billy-idle/pkg/tracker"
)

var punchOpts tracker.Options

// punchCmd represents the busy command
var punchCmd = &cobra.Command{
	Use:     "punch",
	Short:   "Punch the clock - enter or display actual busy time",
	Example: "punch 10h5m 2024-11-07",
	Args:    cobra.MatchAll(cobra.MinimumNArgs(0), cobra.MaximumNArgs(2)),
	Long:    "If no args are provided, the current status for all punched records will be shown",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			if err := punchCreate(cmd.Context(), args); err != nil {
				return err
			}
		}
		punchOpts.Out = rootCmd.OutOrStdout()
		t := tracker.New(&punchOpts)
		return t.PunchReport(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(punchCmd)
	punchCmd.PersistentFlags().StringVarP(&punchOpts.Env, "env", "e", defaultEnv(), "Environment")
	punchCmd.PersistentFlags().StringVarP(&punchOpts.AppRoot, "app-root", "a", defaultAppRoot(), "App Directory e.g. for SQLite DB (defaults to $HOME/.billy-idle/<env>")
	punchCmd.PersistentFlags().DurationVar(&punchOpts.RegBusy, "reg-busy", 7*time.Hour+48*time.Minute, "Regular busy period per day (w/o breaks), report only")
}

// punchCreate creates a new punch record for a particular day
func punchCreate(ctx context.Context, args []string) error {
	var err error
	var day time.Time
	day = time.Now()
	if len(args) == 2 {
		day, err = time.Parse("2006-01-02", args[1])
		if err != nil {
			return err
		}
	}
	dur, err := time.ParseDuration(args[0])
	fmt.Printf("🕰️ Punching busy time=%s for day %s\n", dur, day.Format("2006-01-02 (Monday)"))
	if err != nil {
		return err
	}
	t := tracker.New(&punchOpts)
	return t.UpsertPunchRecord(ctx, dur, day)
}

// punchReport displays the current punch report
