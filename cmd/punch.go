package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/tillkuhn/billy-idle/pkg/tracker"
)

const secsPerHour = 60 * 60

var errArg = errors.New("argument error")

// punchCmd represents the busy command
var punchCmd = &cobra.Command{
	Use:   "punch",
	Short: "Punch busy time",
	Long:  ``,
	RunE: func(_ *cobra.Command, args []string) error {
		var err error
		var day time.Time
		switch len(args) {
		case 0:
			return fmt.Errorf("%w: expected a duration and optional day argument", errArg)
		case 1:
			day = time.Now()
		case 2:
			day, err = time.Parse("2006-01-02", args[1])
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w: to many arguments", errArg)
		}
		fmt.Println("enter busy time=" + args[0])
		t := tracker.New(&trackOpts)
		bt, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		dur := time.Second * time.Duration(bt) * secsPerHour
		if err := t.UpsertPunchRecord(context.Background(), dur, day); err != nil {
			log.Println(err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(punchCmd)
}
