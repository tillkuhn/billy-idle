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

var errArg = errors.New("argument error")

// busyCmd represents the busy command
var busyCmd = &cobra.Command{
	Use:   "busy",
	Short: "Enter busy time",
	Long:  ``,
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("expected a duration argument: %w", errArg)
		}
		fmt.Println("enter busy time=" + args[0])
		if opts.AppDir == "" {
			opts.AppDir = defaultAppDir(opts.Env)
		}
		t := tracker.New(&opts)
		bt, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		if err := t.UpsertBusyRecord(context.Background(), bt*60*60); err != nil {
			log.Println(err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(busyCmd)
}
