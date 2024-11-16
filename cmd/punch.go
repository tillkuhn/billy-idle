package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tillkuhn/billy-idle/pkg/tracker"
)

var punchOpts tracker.Options

// punchCmd represents the busy command
var punchCmd = &cobra.Command{
	Use:     "punch",
	Short:   "Punch the clock - enter actual busy time",
	Example: "punch 10h5m 2024-11-07",
	Args:    cobra.MatchAll(cobra.MinimumNArgs(1), cobra.MaximumNArgs(2)),
	Long:    ``,
	RunE: func(cmd *cobra.Command, args []string) error {
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
		t := tracker.New(&trackOpts)
		if err := t.UpsertPunchRecord(cmd.Context(), dur, day); err != nil {
			return err
		}

		// show status
		recs, err := t.PunchRecords(cmd.Context())
		if err != nil {
			return err
		}
		var spentBusy time.Duration
		for _, r := range recs {
			spentDay := time.Duration(r.BusySecs) * time.Second
			fmt.Printf("🕰️ %s: actual busy time %v\n", r.Day.Format("2006-01-02"), spentDay)
			spentBusy += spentDay
		}
		spentBusy = spentBusy.Round(time.Minute)
		pDays := len(recs)
		expected := time.Duration(pDays) * punchOpts.RegBusy
		overtime := spentBusy - expected
		fmt.Printf("TotalBusy(%dd): %v   AvgPerDay: %v  Expected(%dd*%v): %v   Overtime: %v\n",
			pDays, tracker.FDur(spentBusy), tracker.FDur(spentBusy/time.Duration(pDays)),
			pDays, tracker.FDur(punchOpts.RegBusy), tracker.FDur(expected), tracker.FDur(overtime))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(punchCmd)
	punchCmd.PersistentFlags().DurationVar(&punchOpts.RegBusy, "reg-busy", 7*time.Hour+48*time.Minute, "Regular busy period per day (w/o breaks), report only")
}
