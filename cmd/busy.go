package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// busyCmd represents the busy command
var busyCmd = &cobra.Command{
	Use:   "busy",
	Short: "Enter busy time",
	Long:  ``,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("busy called")
	},
}

func init() {
	rootCmd.AddCommand(busyCmd)
}
