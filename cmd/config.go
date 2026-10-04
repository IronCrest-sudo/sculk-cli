/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"sculk-cli/src/commands/config"

	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config [param] [value]",
	Short: "Read or change sculk-cli configuration.",
	Long: `Read or change sculk-cli configuration.

Running 'sculk config' with no arguments prints every value.

params:
  doMerge        true/false - merge libraries into the current project, or
                 install them as separate datapacks next to it
  author         default author name written into new libraries.json files
  initTemplate   template applied by 'sculk init' ('none' disables it)
  sculk.version  the sculk-cli version recorded in this config

examples:
  sculk config
  sculk config doMerge false
  sculk config author Barden
  sculk config reset
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return config.Main(args)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
