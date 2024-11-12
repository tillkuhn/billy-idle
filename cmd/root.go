package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Useful variables passed with ldflags during build, see goreleaser https://goreleaser.com/cookbooks/using-main.version/
var (
	version = "latest"
	date    = "now"
	// unused: commit, builtBy
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "billy-idle",
	Short: "Simple busy / idle time tracker inspired by the ancient article 'Inactivity and Idle Time on OS X'.",
	Long: `OS X has a timer called HIDIdleTime that tracks the last time you interacted with the computer, e.g. moved the mouse, typed a key, or interacted with the computer.

billy-idle simply queries this value periodically using the ioreg utility that ships with macOS, and matches it against a pre-defined threshold. 
If exceeded, it will create a record for the busy time period in database. This data can later be used as input for time tracking tools or statistics.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.billy-idle.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
