package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "frizzle",
	Short: "Local AWS Event Bus simulator for testing event-driven systems",
	Long: `frizzle is a local Event Bus simulator that lets you test AWS EventBridge
behaviour without needing an AWS account.

It listens for events via a local HTTP API, matches them against configurable
rules (supporting the full EventBridge pattern-matching syntax), logs matched
events to the console, and forwards them to downstream HTTP targets.

Configuration lives in a .frizzle/frizzle.json file in your working directory.
No global config, no AWS credentials needed.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}
