package add

// Separate-pack installation: the `doMerge=false` half of
// `sculk config doMerge`.
//
// Instead of merging a library's files into the project the user is standing
// in, the library is laid down as a sibling pack - which is what Minecraft
// loads as its own datapack/resourcepack. This is the right mode for whole
// engines such as macroEngine, which ship their own load/tick tags, function
// tags and advancements and would otherwise be spliced into the user's pack.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"sculk-cli/src/commands/initProject/create"

	"charm.land/log/v2"

	"github.com/go-git/go-billy/v6"
)

// packContainerNames are the directory names Minecraft (and common server
// setups) load packs from. When sculk is run from inside one of them, a
// separate library really does belong next to the project.
var packContainerNames = map[string]PackKind{
	"datapacks":      KindDatapack,
	"datapack":       KindDatapack,
	"data_packs":     KindDatapack,
	"resourcepacks":  KindResourcepack,
	"resourcepack":   KindResourcepack,
	"resource_packs": KindResourcepack,
}

// SeparateTargetDir returns the directory a library would be installed into as
// a separate pack, without creating anything.
//
// sculk is normally run from inside a pack:
//
//	<world>/datapacks/my_pack          (cwd)
//
// so the sibling is exactly where Minecraft will find it:
//
//	<world>/datapacks/<identifier>
//
// Two safety rules apply:
//
//   - A resourcepack library is redirected to the sibling `resourcepacks/`
//     folder, because a resourcepack inside `datapacks/` would never load.
//   - If the project is NOT inside a pack container, sculk refuses to write
//     into the parent directory (which could be anywhere, including outside
//     the world) and installs into `<project>/libraries/<identifier>` instead,
//     telling the user to move it.
//
// The second return value reports which of the two happened.
func SeparateTargetDir(block libraryBlock) (string, bool, error) {
	cwd := workingDir()
	parent := filepath.Dir(cwd)

	parentName := strings.ToLower(filepath.Base(parent))
	containerKind, inContainer := packContainerNames[parentName]

	if !inContainer {
		// Not inside a pack container: stay inside the project.
		return filepath.Join(cwd, "libraries", SanitizeFolderName(block.Identifier)), false, nil
	}

	if block.packKind() == KindResourcepack && containerKind == KindDatapack {
		// <world>/datapacks -> <world>/resourcepacks
		parent = filepath.Join(filepath.Dir(parent), "resourcepacks")
	}

	return filepath.Join(parent, SanitizeFolderName(block.Identifier)), true, nil
}

// InstallAsSeparatePack writes the library out as its own pack and returns the
// directory it was written to.
func InstallAsSeparatePack(r ResolvedLibrary) (string, error) {
	target, inContainer, err := SeparateTargetDir(r.Block)
	if err != nil {
		return "", err
	}

	if !inContainer {
		log.Printf("⚠ This project is not inside a 'datapacks/' folder, so the pack cannot be placed beside it.")
		log.Printf("⚠ Installed into %s instead - move it into your world's %s folder (or symlink it) for Minecraft to load it.",
			target, containerFolderFor(r.Block))
	}

	// Never clobber an existing pack silently.
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		if entries, err := os.ReadDir(target); err == nil && len(entries) > 0 {
			return "", fmt.Errorf(
				"%s already exists and is not empty; remove it first, or run 'sculk config doMerge true' to merge instead",
				target,
			)
		}
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		return "", err
	}

	wrotePackMcmeta := false
	if err := copyTree(r.Fs, r.Root, target, &wrotePackMcmeta); err != nil {
		return "", err
	}

	// A standalone pack needs a pack.mcmeta or Minecraft ignores it.
	if !wrotePackMcmeta {
		if err := writeFallbackPackMcmeta(target, r); err != nil {
			log.Printf("⚠ Could not write pack.mcmeta for '%s': %v", r.Block.Identifier, err)
		}
	}

	// Keep a libraries.json inside the pack too, so that sculk can inspect or
	// update it on its own.
	if _, err := os.Stat(filepath.Join(r.Root, "libraries.json")); err != nil {
		if err := writePackLibrariesJson(target, r); err != nil {
			log.Printf("⚠ Could not write libraries.json for '%s': %v", r.Block.Identifier, err)
		}
	}

	return target, nil
}

// containerFolderFor names the Minecraft folder a pack of this kind belongs in.
func containerFolderFor(block libraryBlock) string {
	if block.packKind() == KindResourcepack {
		return "resourcepacks/"
	}
	return "datapacks/"
}

// copyTree writes every file under src into dst. Nothing is skipped: a
// separate pack keeps its own pack.mcmeta, README and LICENSE.
func copyTree(fs billy.Filesystem, src string, dst string, wrotePackMcmeta *bool) error {
	entries, err := fs.ReadDir(src)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return err
			}
			if err := copyTree(fs, srcPath, dstPath, wrotePackMcmeta); err != nil {
				return err
			}
			continue
		}

		if entry.Name() == "pack.mcmeta" {
			*wrotePackMcmeta = true
		}

		if err := copyFile(fs, srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(fs billy.Filesystem, src string, dst string) error {
	in, err := fs.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// writeFallbackPackMcmeta generates a pack.mcmeta for libraries that do not
// ship one. The pack format comes from the library's declared game version.
func writeFallbackPackMcmeta(target string, r ResolvedLibrary) error {
	isRp := r.Block.packKind() == KindResourcepack

	kind := "datapack"
	if isRp {
		kind = "resourcepack"
	}
	packFormat, exact := create.PackFormatFor(r.Meta.GameVersion, kind)
	if !exact {
		log.Printf("⚠ No known pack format for Minecraft %s; using %d. Check %s before loading.",
			r.Meta.GameVersion, packFormat, filepath.Join(target, "pack.mcmeta"))
	}

	content := create.PackMcmetaFileContent{
		PackData: create.Pack{
			Pack_format: packFormat,
			Description: fmt.Sprintf("%s (installed separately by sculk)", r.Block.Identifier),
		},
	}
	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(target, "pack.mcmeta"), data, 0644)
}

// writePackLibrariesJson drops a libraries.json into a separately installed
// pack so that it describes itself.
func writePackLibrariesJson(target string, r ResolvedLibrary) error {
	author := r.Meta.Author
	if author == "" {
		author = "unknown"
	}

	meta := create.LibrariesDotJson{
		Name:        r.Block.Identifier,
		Author:      author,
		Version:     r.Meta.Version,
		GameVersion: r.Meta.GameVersion,
		Libraries:   []create.Library{},
	}
	data, err := json.MarshalIndent(meta, "", "   ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(target, "libraries.json"), data, 0644)
}
