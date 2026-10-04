/*
Copyright © 2026 barden <theofficialbarden@gmail.com>
*/
package cmd

import (
	"sculk-cli/src/commands/update"

	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update [...libraryName]",
	Short: "Update specific libraries, or all of them, to the newest version.",
	Long: `Update libraries to the newest published version.

Versions are compared with sculk's full constraint engine, so a library pinned
with 'sculk add lib@^1.0.0' stays inside that range. With no arguments every
library in libraries.json is updated.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return update.Main(args)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
