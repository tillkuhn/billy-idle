package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/tillkuhn/billy-idle/pkg/tracker"
)

// punchCmd represents the busy command
var punchCmd = &cobra.Command{
	Use:     "punch",
	Short:   "Punch the clock - enter actual busy time",
	Example: "punch 10h5m 2024-11-07",
	Args:    cobra.MatchAll(cobra.MinimumNArgs(1), cobra.MaximumNArgs(2)),
	Long:    ``,
	RunE: func(_ *cobra.Command, args []string) error {
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
		if err := t.UpsertPunchRecord(context.Background(), dur, day); err != nil {
			log.Println(err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(punchCmd)
}
