/*
Copyright © 2026 barden <theofficialbarden@gmail.com>
*/
package cmd

import (
	"sculk-cli/src/commands/install"

	"github.com/spf13/cobra"
)

var installIgnoreVersionMismatch bool

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install all libraries specified in libraries.json",
	Long: `Install every library recorded in libraries.json.

This is the command to run after cloning someone else's project: it fetches
the same libraries, at the recorded versions and refs, and lays them down the
same way (merged, or as separate packs) that they were originally added.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return install.Main(installIgnoreVersionMismatch)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVar(&installIgnoreVersionMismatch, "ignore", false,
		"Ignore game version mismatch checking and install anyway.")
}
