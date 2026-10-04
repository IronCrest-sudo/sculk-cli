/*
Copyright © 2026 barden <theofficialbarden@gmail.com>
*/
package cmd

import (
	"sculk-cli/src/commands/list"

	"github.com/spf13/cobra"
)

var (
	listAsJSON    bool
	listInstalled bool
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list [...filter]",
	Aliases: []string{"ls", "search"},
	Short:   "List the libraries sculk knows about, and which are installed.",
	Long: `List the libraries sculk knows about, and which of them are installed in
the current project.

Any extra arguments narrow the list by matching the identifier, description,
author, source or kind.

examples:
  sculk list
  sculk list string
  sculk list --installed
  sculk list --json > website/src/data/libraries.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Re-emit the flags as tokens so that the command package stays free of
		// cobra and can be tested on its own.
		tokens := make([]string, 0, len(args)+2)
		if listAsJSON {
			tokens = append(tokens, "--json")
		}
		if listInstalled {
			tokens = append(tokens, "--installed")
		}
		tokens = append(tokens, args...)
		return list.Main(tokens)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&listAsJSON, "json", "j", false,
		"Machine readable output; this is what the website's library data file is generated from.")
	listCmd.Flags().BoolVarP(&listInstalled, "installed", "i", false,
		"Only show libraries already present in libraries.json.")
}
