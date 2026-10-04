package uninstall

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"sculk-cli/src/commands/add"
	"sculk-cli/src/commands/initProject/create"

	"charm.land/log/v2"
	"github.com/go-git/go-billy/v6"
)

// Similar to `sculk add`, walk to every file, then delete instead of merging.
// Libraries that were installed separately (doMerge=false) are removed by
// deleting the pack directory sculk created for them.
func Main(args []string) error {
	libraryIdentifiers := args

	// No identifiers given -> uninstall everything in libraries.json.
	if len(libraryIdentifiers) == 0 {
		for _, lib := range add.ReadLocalLibrariesJson().Libraries {
			libraryIdentifiers = append(libraryIdentifiers, lib.Identifier)
		}
		if len(libraryIdentifiers) == 0 {
			log.Printf("ℹ Nothing to uninstall, libraries.json is empty.")
			return nil
		}
	}

	var failed []string
	for _, libraryIdentifier := range libraryIdentifiers {
		if err := UninstallLibrary(libraryIdentifier); err != nil {
			log.Printf("⚠ Could not uninstall '%s': %v", libraryIdentifier, err)
			failed = append(failed, libraryIdentifier)
		}
	}

	log.Printf("Note: Please delete any empty directories manually.")

	if len(failed) > 0 {
		return errJoin(failed)
	}
	return nil
}

type joinError struct{ items []string }

func (e joinError) Error() string  { return "failed to uninstall: " + strings.Join(e.items, ", ") }
func errJoin(items []string) error { return joinError{items} }

// UninstallLibrary removes one library and its libraries.json record.
func UninstallLibrary(libraryIdentifier string) error {
	record, installed := add.FindInstalled(libraryIdentifier)

	// Separately installed packs are simply their own directory.
	if installed && record.Install == create.InstallSeparate {
		return removeSeparatePack(libraryIdentifier, record)
	}

	// Merged libraries: re-fetch the source and delete exactly what it added.
	spec, err := add.ParseSpec(libraryIdentifier)
	if err != nil {
		return err
	}
	// Keep the pinned ref so the deletion matches what was installed.
	if installed && record.Ref != "" {
		spec, err = add.ParseSpec(libraryIdentifier + "@" + record.Ref)
		if err != nil {
			return err
		}
	}

	resolved, err := add.Resolve(spec)
	if err != nil {
		return err
	}

	targetDir, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := TraverseSourceCode(resolved.Fs, resolved.Root, targetDir); err != nil {
		return err
	}

	return add.RemoveFromLibrariesJson(libraryIdentifier)
}

// removeSeparatePack deletes the sibling pack directory sculk created.
func removeSeparatePack(libraryIdentifier string, record create.Library) error {
	block := add.VerifyLibraryIntegrity(libraryIdentifier)

	dir, _, err := add.SeparateTargetDir(block)
	if err != nil {
		return err
	}

	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		log.Printf("🚮 Removing separate pack %s", dir)
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	} else {
		log.Printf("⚠ Separate pack directory not found at %s (already removed?)", dir)
	}
	_ = record

	return add.RemoveFromLibrariesJson(libraryIdentifier)
}

func TraverseSourceCode(fs billy.Filesystem, currentPath string, targetDir string) error {
	files, err := fs.ReadDir(currentPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		// store memory path and local path
		memoryPath := filepath.Join(currentPath, file.Name())
		localPath := filepath.Join(targetDir, file.Name())

		if file.IsDir() {
			// if found directory, go inside that directory and recurse.
			if err := TraverseSourceCode(fs, memoryPath, localPath); err != nil {
				return err
			}

		} else {

			// blacklisted filenames
			blacklistedFiles := []string{
				"libraries.json",
				"LICENSE",
				"pack.mcmeta",
				"README.md",
			}

			deleted, err := handleFileDeletion(fs, memoryPath, localPath, blacklistedFiles)
			if err != nil {
				return err
			}
			if deleted {
				log.Printf("🚮 Deleted file '%s'.", file.Name())
			}
		}
	}
	return nil
}

// handleFileDeletion removes what one library file contributed to the
// project. It reports whether anything was actually removed, so that skipped
// (blacklisted or absent) files are not logged as deletions.
func handleFileDeletion(fs billy.Filesystem, sourcePath string, destinationPath string, blacklist []string) (bool, error) {

	// open source file (contains the lines/entries to remove)
	srcFile, err := fs.Open(sourcePath)
	if err != nil {
		log.Error("⚠ Source File Doesn't Exist.")
		return false, err
	}
	defer srcFile.Close()

	// nothing to delete from if destination doesn't exist
	fileInfo, err := os.Stat(destinationPath)
	if err != nil {
		log.Printf("⚠ Destination File Doesn't Exist, nothing to delete: %s", destinationPath)
		return false, nil
	}

	if isBlacklisted(fileInfo.Name(), blacklist) {
		log.Printf("🚫 %s is blacklisted, skipping deletion", fileInfo.Name())
		return false, nil
	}

	fileExtension := strings.TrimPrefix(filepath.Ext(fileInfo.Name()), ".")

	sourceContent, err := io.ReadAll(srcFile)
	if err != nil {
		return false, err
	}

	rootDir, err := os.Getwd()
	if err != nil {
		return false, err
	}

	if fileExtension == "json" {

		if fileInfo.Name() == "load.json" || fileInfo.Name() == "tick.json" {
			log.Printf("🗑 Removing entries from %s", fileInfo.Name())

			var sourceTagContent add.FunctionTag
			var existingTagContent add.FunctionTag

			existingContent, err := os.ReadFile(destinationPath)
			if err != nil {
				return false, err
			}

			if err := json.Unmarshal(sourceContent, &sourceTagContent); err != nil {
				return false, err
			}
			if err := json.Unmarshal(existingContent, &existingTagContent); err != nil {
				return false, err
			}

			// build a lookup of values to remove
			toRemove := make(map[string]bool, len(sourceTagContent.Values))
			for _, v := range sourceTagContent.Values {
				toRemove[v] = true
			}

			filtered := existingTagContent.Values[:0]
			for _, v := range existingTagContent.Values {
				if !toRemove[v] {
					filtered = append(filtered, v)
				}
			}
			existingTagContent.Values = filtered

			// if nothing left, delete the file entirely (and any now-empty parent dirs)
			if len(existingTagContent.Values) == 0 {
				log.Printf("🗑 %s is now empty, deleting file", fileInfo.Name())
				if err := os.Remove(destinationPath); err != nil {
					return false, err
				}
				return true, removeEmptyDirs(filepath.Dir(destinationPath), rootDir, blacklist)
			}

			combined, err := json.MarshalIndent(existingTagContent, "", "  ")
			if err != nil {
				return false, err
			}
			return true, os.WriteFile(destinationPath, combined, 0644)
		} else {
			return true, os.Remove(destinationPath)
		}
	}

	// generic line-based deletion
	existingData, err := os.ReadFile(destinationPath)
	if err != nil {
		return false, err
	}

	// lines to remove, exact match
	toRemove := make(map[string]bool)
	for _, line := range strings.Split(string(sourceContent), "\n") {
		toRemove[line] = true
	}

	var kept []string
	for _, line := range strings.Split(string(existingData), "\n") {
		if !toRemove[line] {
			kept = append(kept, line)
		}
	}

	remaining := strings.TrimSpace(strings.Join(kept, "\n"))

	// if file becomes empty, delete it entirely (and any now-empty parent dirs)
	if remaining == "" {
		log.Printf("🗑 %s is now empty, deleting file", fileInfo.Name())
		if err := os.Remove(destinationPath); err != nil {
			return false, err
		}
		return true, removeEmptyDirs(filepath.Dir(destinationPath), rootDir, blacklist)
	}

	return true, os.WriteFile(destinationPath, []byte(remaining+"\n"), 0644)
}

func isBlacklisted(name string, blacklist []string) bool {
	for _, b := range blacklist {
		if b == name {
			return true
		}
	}
	return false
}

func removeEmptyDirs(dir string, root string, blacklist []string) error {
	root = filepath.Clean(root)

	for {
		dir = filepath.Clean(dir)

		if dir == root || dir == "." || dir == string(filepath.Separator) {
			return nil
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil
		}

		if isBlacklisted(filepath.Base(dir), blacklist) {
			log.Printf("🚫 %s is blacklisted, stopping cleanup here", dir)
			return nil
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		if len(entries) > 0 {
			return nil
		}

		if err := os.Remove(dir); err != nil {
			return nil
		}
		dir = filepath.Dir(dir)
	}
}
