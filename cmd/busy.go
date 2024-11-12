package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
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
		fmt.Println("enter budy time=" + args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(busyCmd)
}
