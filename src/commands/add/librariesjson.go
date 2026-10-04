package add

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sculk-cli/src/commands/initProject/create"

	"charm.land/log/v2"
)

// librariesJsonPath is the absolute path of this project's libraries.json.
func librariesJsonPath() string {
	return filepath.Join(workingDir(), "libraries.json")
}

// AddToLibrariesJsonRecord records an already resolved library in the local
// libraries.json. This is the fast path: nothing is downloaded again.
func AddToLibrariesJsonRecord(resolved ResolvedLibrary) error {
	record := libraryRecord(resolved)

	log.Printf("🚧 Adding '%s' library to libraries.json ...", record.Identifier)

	jsonPath := librariesJsonPath()
	if _, err := os.Stat(jsonPath); err != nil {
		return fmt.Errorf("no libraries.json here - run 'sculk init' first (%w)", err)
	}

	librariesJson := ReadLocalLibrariesJson()

	for i, lib := range librariesJson.Libraries {
		if lib.Identifier == record.Identifier {
			log.Printf("⚠ Library '%s' already exists in libraries.json", record.Identifier)
			return nil
		}
		// Self-heal entries that were written with a browser URL instead of
		// the registry identifier.
		if id, ok := RegistryIdentifierForURL(lib.Identifier); ok && strings.EqualFold(id, record.Identifier) {
			log.Printf("🩹 Rewriting '%s' in libraries.json as '%s'.", lib.Identifier, record.Identifier)
			librariesJson.Libraries[i] = record
			return writeLibrariesJson(jsonPath, librariesJson)
		}
	}

	librariesJson.Libraries = append(librariesJson.Libraries, record)
	return writeLibrariesJson(jsonPath, librariesJson)
}

// AddToLibrariesJson resolves an identifier and records it. Kept for call sites
// that only have a raw identifier.
func AddToLibrariesJson(libraryIdentifier string) error {
	spec, err := ParseSpec(libraryIdentifier)
	if err != nil {
		return err
	}
	resolved, err := Resolve(spec)
	if err != nil {
		return err
	}
	resolved.Block = VerifyLibraryIntegrity(spec.Identifier)
	return AddToLibrariesJsonRecord(resolved)
}

// UpdateLibrariesJsonEntry replaces (or inserts) the record for identifier.
// `sculk update` uses this so that a bumped version is reflected without a
// remove/add round trip.
func UpdateLibrariesJsonEntry(record create.Library) error {
	jsonPath := librariesJsonPath()
	if _, err := os.Stat(jsonPath); err != nil {
		return fmt.Errorf("no libraries.json here - run 'sculk init' first (%w)", err)
	}

	librariesJson := ReadLocalLibrariesJson()

	replaced := false
	for i := range librariesJson.Libraries {
		if librariesJson.Libraries[i].Identifier == record.Identifier {
			librariesJson.Libraries[i] = record
			replaced = true
			break
		}
	}
	if !replaced {
		librariesJson.Libraries = append(librariesJson.Libraries, record)
	}

	return writeLibrariesJson(jsonPath, librariesJson)
}

func RemoveFromLibrariesJson(libraryIdentifier string) error {
	log.Printf("🚧 Removing '%s' library from libraries.json ...", libraryIdentifier)

	jsonPath := librariesJsonPath()
	if _, err := os.Stat(jsonPath); err != nil {
		log.Printf("⚠ No libraries.json here, nothing to remove from.")
		return nil
	}

	librariesJson := ReadLocalLibrariesJson()

	found := false
	filtered := librariesJson.Libraries[:0]
	for _, lib := range librariesJson.Libraries {
		if lib.Identifier == libraryIdentifier {
			found = true
			continue
		}
		filtered = append(filtered, lib)
	}

	if !found {
		log.Printf("⚠ Library '%s' not found in libraries.json", libraryIdentifier)
		return nil
	}

	librariesJson.Libraries = filtered

	log.Printf("🗑 Removed from libraries.json")
	return writeLibrariesJson(jsonPath, librariesJson)
}

// FindInstalled returns the libraries.json record for an identifier.
func FindInstalled(libraryIdentifier string) (create.Library, bool) {
	for _, lib := range ReadLocalLibrariesJson().Libraries {
		if lib.Identifier == libraryIdentifier {
			return lib, true
		}
	}
	return create.Library{}, false
}

func writeLibrariesJson(path string, data create.LibrariesDotJson) error {
	if data.Libraries == nil {
		data.Libraries = []create.Library{}
	}
	combined, err := json.MarshalIndent(data, "", "   ")
	if err != nil {
		return err
	}
	combined = append(combined, '\n')
	return os.WriteFile(path, combined, 0644)
}
