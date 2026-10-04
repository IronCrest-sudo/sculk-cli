// To update:
//  1. Check whether a newer version exists (using the shared version package,
//     so ranges and pre-releases compare correctly).
//  2. If it does, uninstall the current version and install the new one.
package update

import (
	"os"
	"strings"

	"sculk-cli/src/commands/add"
	"sculk-cli/src/commands/config"
	"sculk-cli/src/commands/initProject/create"
	"sculk-cli/src/commands/uninstall"
	"sculk-cli/src/version"

	"charm.land/log/v2"
)

func Main(args []string) error {
	if _, err := os.Stat("libraries.json"); err != nil {
		log.Printf("⚠ No libraries.json here - run 'sculk init' first.")
		return err
	}

	identifiers := args

	// No identifiers given -> update everything in libraries.json.
	if len(identifiers) == 0 {
		for _, lib := range add.ReadLocalLibrariesJson().Libraries {
			identifiers = append(identifiers, lib.Identifier)
		}
		if len(identifiers) == 0 {
			log.Printf("ℹ Nothing to update, libraries.json is empty.")
			return nil
		}
	}

	var failed []string
	for _, identifier := range identifiers {
		if err := updateOne(identifier); err != nil {
			log.Printf("⚠ Could not update '%s': %v", identifier, err)
			failed = append(failed, identifier)
		}
	}

	if len(failed) > 0 {
		return errJoin(failed)
	}
	return nil
}

type joinError struct{ items []string }

func (e joinError) Error() string  { return "failed to update: " + strings.Join(e.items, ", ") }
func errJoin(items []string) error { return joinError{items} }

// updateOne brings a single library up to date.
func updateOne(libraryIdentifier string) error {
	installedRecord, installed := add.FindInstalled(libraryIdentifier)

	// Re-resolve using the same pinned ref the user asked for, so that
	// `sculk add lib@1.0.0` stays on the 1.0.0 line and only picks up
	// revisions published under that ref.
	spec, err := add.ParseSpec(libraryIdentifier)
	if err != nil {
		return err
	}
	if installed && installedRecord.Ref != "" {
		spec, err = add.ParseSpec(libraryIdentifier + "@" + installedRecord.Ref)
		if err != nil {
			return err
		}
	}

	resolved, err := add.Resolve(spec)
	if err != nil {
		return err
	}
	resolved.Block = add.VerifyLibraryIntegrity(libraryIdentifier)

	newVersion := resolved.Meta.Version
	oldVersion := "0.0.0"
	if installed {
		oldVersion = installedRecord.Version
	}

	// A newer source version, or a first-time record, both count as work.
	isOutdated := !installed || version.CompareStrings(newVersion, oldVersion) > 0

	if !isOutdated {
		log.Printf("🍀 Library '%s' is up-to-date (v. %s).", libraryIdentifier, newVersion)
		return nil
	}

	log.Printf("🍁 Library '%s' is outdated. Updating [%s -> %s] ...", libraryIdentifier, oldVersion, newVersion)

	if installed {
		if err := uninstall.UninstallLibrary(libraryIdentifier); err != nil {
			return err
		}
	}

	if err := add.InstallLibrary(spec, false); err != nil {
		return err
	}

	log.Printf("🍀 Library '%s' has been updated [%s -> %s].", libraryIdentifier, oldVersion, newVersion)
	return nil
}

// InstallModeOf is exposed so callers can report how a library is laid down.
func InstallModeOf(record create.Library) string {
	if record.Install != "" {
		return record.Install
	}
	if config.ShouldMerge() {
		return create.InstallMerge
	}
	return create.InstallSeparate
}
