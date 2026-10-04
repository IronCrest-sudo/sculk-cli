/*
Copyright © 2026 barden <theofficialbarden@gmail.com>
*/
package cmd

import (
	"sculk-cli/src/commands/uninstall"

	"github.com/spf13/cobra"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall [...libraryName]",
	Short: "Uninstall specific libraries, or all of them when none are named.",
	Long: `Uninstall libraries from the current project.

Merged libraries have exactly the files they added removed, and their entries
in load.json / tick.json are dropped. Libraries that were installed as
separate packs have their pack directory deleted.

With no arguments every library in libraries.json is removed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return uninstall.Main(args)
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
