// `sculk install` - install every library recorded in libraries.json.
//
// This is the command a collaborator runs after cloning a project: they get
// the same libraries, at the same versions, laid down the same way.
package install

import (
	"fmt"
	"os"
	"strings"

	"sculk-cli/src/commands/add"
	"sculk-cli/src/commands/initProject/create"

	"charm.land/log/v2"
)

// Main installs every library listed in libraries.json.
//
// ignoreVersionMismatch maps to `sculk install --ignore`.
func Main(ignoreVersionMismatch bool) error {
	filePath := "libraries.json"
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("no libraries.json here - run 'sculk init' first, or 'sculk makelib' to adopt an existing pack")
	}

	project := add.ReadLocalLibrariesJson()
	if len(project.Libraries) == 0 {
		log.Printf("ℹ libraries.json lists no libraries, nothing to install.")
		return nil
	}

	log.Printf("📥 Installing %d libraries for Minecraft %s ...", len(project.Libraries), project.GameVersion)

	var failed []string
	for _, lib := range project.Libraries {
		if err := installOne(lib, ignoreVersionMismatch); err != nil {
			log.Printf("⚠ Could not install '%s': %v", lib.Identifier, err)
			failed = append(failed, lib.Identifier)
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to install: %s", strings.Join(failed, ", "))
	}

	log.Printf("✅ All libraries installed.")
	return nil
}

// installOne installs a single recorded library, honouring the ref and the
// install mode that were recorded when it was first added.
func installOne(lib create.Library, ignoreVersionMismatch bool) error {
	// Re-pin the exact ref that was recorded, so a shared libraries.json does
	// not silently drift to a newer branch.
	target := lib.Identifier
	if lib.Ref != "" {
		target = lib.Identifier + "@" + lib.Ref
	}

	spec, err := add.ParseSpec(target)
	if err != nil {
		return err
	}

	mode := lib.Install
	if mode == "" {
		mode = add.DefaultInstallMode()
	}

	log.Printf("📦 %s  (v%s, %s mode)", spec.String(), lib.Version, mode)
	return add.InstallRecordedLibrary(spec, mode, ignoreVersionMismatch)
}
