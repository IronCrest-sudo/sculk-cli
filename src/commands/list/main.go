// `sculk list` - show what is available and what is installed.
//
// With --json it emits the registry in a machine readable form; the website's
// library data file is generated from exactly this output so that the CLI and
// the site can never disagree:
//
//	sculk list --json > website/src/data/libraries.json
package list

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"sculk-cli/src/commands/add"
	"sculk-cli/src/commands/initProject/create"
)

// Entry is one row of `sculk list --json`.
type Entry struct {
	Identifier  string `json:"identifier"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
	Source      string `json:"source"`
	Subdir      string `json:"subdir,omitempty"`
	Kind        string `json:"kind"`
	Homepage    string `json:"homepage,omitempty"`
	Notice      string `json:"notice,omitempty"`
	// Installed is the locally installed version, "" when not installed.
	Installed string `json:"installed,omitempty"`
}

// Entries builds the registry listing, annotated with what is installed
// locally. It is the single source of truth for both the CLI table and the
// JSON export.
func Entries() []Entry {
	installed := map[string]string{}
	for _, lib := range add.ReadLocalLibrariesJson().Libraries {
		installed[lib.Identifier] = lib.Version
	}

	out := make([]Entry, 0, 16)
	for _, block := range add.ListApprovedLibraries() {
		out = append(out, Entry{
			Identifier:  block.Identifier,
			Description: block.Description,
			Author:      block.Author,
			Source:      block.Source,
			Subdir:      block.Subdir,
			Kind:        string(block.KindOrDefault()),
			Homepage:    block.Homepage,
			Notice:      block.Notice,
			Installed:   installed[block.Identifier],
		})
	}
	return out
}

// Main implements `sculk list [--json] [--installed] [filter...]`.
func Main(args []string) error {
	asJSON := false
	installedOnly := false
	var filters []string

	for _, a := range args {
		switch strings.ToLower(a) {
		case "--json", "-j", "json":
			asJSON = true
		case "--installed", "-i", "installed":
			installedOnly = true
		default:
			filters = append(filters, strings.ToLower(a))
		}
	}

	entries := Entries()

	if installedOnly {
		var kept []Entry
		for _, e := range entries {
			if e.Installed != "" {
				kept = append(kept, e)
			}
		}
		entries = kept
	}

	if len(filters) > 0 {
		var kept []Entry
		for _, e := range entries {
			if matchesFilters(e, filters) {
				kept = append(kept, e)
			}
		}
		entries = kept
	}

	if asJSON {
		return writeJSON(entries)
	}
	writeTable(entries, installedOnly)
	return nil
}

func matchesFilters(e Entry, filters []string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		e.Identifier, e.Description, e.Author, e.Source, e.Subdir, e.Kind,
	}, " "))
	for _, f := range filters {
		if !strings.Contains(haystack, f) {
			return false
		}
	}
	return true
}

func writeJSON(entries []Entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = os.Stdout.Write(data)
	return err
}

func writeTable(entries []Entry, installedOnly bool) {
	if len(entries) == 0 {
		if installedOnly {
			fmt.Printf("\nNo libraries installed yet. Try 'sculk add id-system'.\n\n")
		} else {
			fmt.Printf("\nNo libraries match.\n\n")
		}
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "\n  IDENTIFIER\tKIND\tAUTHOR\tINSTALLED\tDESCRIPTION")
	for _, e := range entries {
		state := "-"
		if e.Installed != "" {
			state = e.Installed
		}
		desc := e.Description
		if len(desc) > 62 {
			desc = desc[:59] + "..."
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\n", e.Identifier, shortKind(e.Kind), e.Author, state, desc)
	}
	w.Flush()

	fmt.Printf("\n  install one with:  sculk add <identifier>\n")
	fmt.Printf("  pin a version with: sculk add <identifier>@1.0.0/26.2\n\n")
}

func shortKind(kind string) string {
	switch kind {
	case "resourcepack":
		return "rp"
	case "datapack":
		return "dp"
	}
	return kind
}

// InstalledLibraries returns the local libraries.json entries, for callers that
// want the raw records rather than the registry view.
func InstalledLibraries() []create.Library {
	return add.ReadLocalLibrariesJson().Libraries
}
