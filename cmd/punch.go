package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"

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
		if len(args) < 1 {
			return fmt.Errorf("expected a duration argument: %w", errArg)
		}
		fmt.Println("enter busy time=" + args[0])
		t := tracker.New(&trackOpts)
		bt, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		if err := t.UpsertPunchRecord(context.Background(), bt*secsPerHour); err != nil {
			log.Println(err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(punchCmd)
}
