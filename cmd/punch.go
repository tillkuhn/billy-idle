package cmd

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/spf13/cobra"
	"github.com/tillkuhn/billy-idle/pkg/tracker"
)

var punchOpts tracker.Options

var shortDate = regexp.MustCompile(`^\d+-\d+$`)

// punchCmd represents the busy command
var punchCmd = &cobra.Command{
	Use:     "punch",
	Short:   "Punch the clock - enter or display actual busy time",
	Example: "punch 10h5m 2024-11-07",
	Args:    cobra.MatchAll(cobra.MinimumNArgs(0), cobra.MaximumNArgs(3)),
	Long: `If no args are provided, the current status for all punched records will be shown
If args are provided, the first argument is considered as the actual duration for the current day.
If 2nd are is provided, the 2nd arg is considered as the day in YYYY-MM-DD format.
If the 3rd arg is provided, it is considered to be the planned duration (which defaults to the options value).
The following examples records 2 hours and 5m for Dec. 24th 2024 with a planned time of 3hours and 54m.  

punch 2h5m 2024-12-24 3h54m`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			if err := punchCreate(cmd.Context(), args); err != nil {
				return err
			}
		}
		punchOpts.Out = rootCmd.OutOrStdout()
		t := tracker.New(&punchOpts)
		return t.PunchReport(cmd.Context()) // always show current report, even in create mode
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
	// if 2nd arg exists, consider it to be the target day, otherwise assume "today"
	if len(args) >= 2 {
		// todo check if numeric arg (relative number of days) first
		// var digitCheck = regexp.MustCompile(`^[-+]?[0-9]+$`)
		// num := "+1212"
		// fmt.Println(digitCheck.MatchString(num))
		// i, _ := strconv.ParseInt(num, 0, 64)

		// if year is omitted, use current year as prefix
		if shortDate.MatchString(args[1]) {
			args[1] = fmt.Sprintf("%s-%s", time.Now().Format("2006"), args[1])
		}
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

	// if planned duration is provided as 3rd arg, use it instead of the default derived from options
	plannedDur := punchOpts.RegBusy
	if len(args) >= 3 {
		plannedDur, err = time.ParseDuration(args[2])
		if err != nil {
			return err
		}
	}

	return t.UpsertPunchRecordWithPlannedDuration(ctx, dur, day, plannedDur)
}

// punchReport displays the current punch report
