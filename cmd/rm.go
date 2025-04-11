package cmd

import (
	"strconv"

	"github.com/tillkuhn/billy-idle/pkg/tracker"

	"github.com/spf13/cobra"
)

var rmOpts tracker.Options

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:     "rm",
	Short:   "Remove a busy / idle record from DB",
	Example: "rm 1234",
	Args:    cobra.ExactArgs(1),
	Long:    `Sometimes billy tracks something you don't want to be tracked, so rm allows you to delete records by id '.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rmOpts.Out = rootCmd.OutOrStdout()
		t := tracker.New(&rmOpts)
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		return t.RemoveRecord(cmd.Context(), id) // always show current report, even in create mode
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.PersistentFlags().StringVarP(&rmOpts.Env, "env", "e", defaultEnv(), "Environment")
	rmCmd.PersistentFlags().StringVarP(&rmOpts.AppRoot, "app-root", "a", defaultAppRoot(), "App Directory e.g. for SQLite DB (defaults to $HOME/.billy-idle/<env>")
}
