package add

// Steps
// 1. Parse the argument into a LibrarySpec (identifier[@version[/gameVersion]])
// 2. Map the identifier through the registry, resolve it to a git ref
// 3. Clone the repository (or just the registered subdirectory) into memory
// 4. Either merge it into the current project, or lay it down as a separate
//    pack next to it - whichever `sculk config doMerge` says

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"sculk-cli/src/commands/config"
	"sculk-cli/src/commands/initProject/create"
	"sculk-cli/src/version"

	"charm.land/log/v2"

	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/memfs"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/storage/memory"
)

// ResolvedLibrary is one library, fully resolved and sitting in memory, ready
// to be written to disk.
type ResolvedLibrary struct {
	// Spec is what the user typed.
	Spec LibrarySpec
	// Block is the registry entry (or a synthesized one for direct URLs).
	Block libraryBlock
	// RepoURL is the git remote it came from.
	RepoURL string
	// Root is the path prefix inside Fs that holds the pack, e.g. "/" or
	// "/archived/macroEngine-Datapack-v26.4". It always starts and ends
	// without a trailing slash, except for "/" itself.
	Root string
	// Fs is the in-memory checkout.
	Fs billy.Filesystem
	// Meta is the library's own libraries.json, synthesized when absent.
	Meta create.LibrariesDotJson
	// Ref is the branch/tag it was resolved from; HasRef is false when the
	// repository's default branch was used.
	Ref    RemoteRef
	HasRef bool
	// Mode is "merge" or "separate"; it is filled in by the installer, not by
	// Resolve, because `sculk install` honours the mode recorded in
	// libraries.json rather than the current config.
	Mode string
}

// InstallMode reports how this library will be laid down.
func (r ResolvedLibrary) InstallMode() string {
	if r.Mode != "" {
		return r.Mode
	}
	return DefaultInstallMode()
}

// DefaultInstallMode is the mode a fresh install uses, straight from
// `sculk config doMerge`.
func DefaultInstallMode() string {
	if config.ShouldMerge() {
		return create.InstallMerge
	}
	return create.InstallSeparate
}

// RefLabel is the human readable ref, "" for the default branch.
func (r ResolvedLibrary) RefLabel() string {
	if !r.HasRef {
		return ""
	}
	return r.Ref.FullName()
}

// Main is the entry point of `sculk add <identifier|url>[@spec] ...`.
func Main(ignoreVersionMismatch bool, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no library given; try 'sculk add id-system' or 'sculk list'")
	}

	var failed []string

	for i := range args {
		spec, err := ParseSpec(args[i])
		if err != nil {
			log.Printf("⚠ %v", err)
			failed = append(failed, args[i])
			continue
		}

		if err := InstallLibrary(spec, ignoreVersionMismatch); err != nil {
			log.Printf("⚠ Could not install '%s': %v", spec.String(), err)
			failed = append(failed, spec.String())
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to install: %s", strings.Join(failed, ", "))
	}
	return nil
}

// InstallLibrary resolves and installs a single library, using the mode from
// `sculk config doMerge`.
func InstallLibrary(spec LibrarySpec, ignoreVersionMismatch bool) error {
	return InstallLibraryWithMode(spec, "", ignoreVersionMismatch)
}

// InstallLibraryWithMode installs a library in an explicit mode
// ("merge"/"separate"). An empty mode means "whatever the config says", which
// is what `sculk add` wants; `sculk install` passes the mode recorded in
// libraries.json so that a shared libraries.json reproduces the same layout on
// every machine.
func InstallLibraryWithMode(spec LibrarySpec, mode string, ignoreVersionMismatch bool) error {
	// Registry lookup happens on the bare identifier so that
	// "id-system@1.0.0" and "id-system" hit the same entry.
	spec.Identifier = lookupIdentifier(spec)

	if IsPreinstalled(spec.Identifier) {
		return nil
	}

	return InstallRecordedLibrary(spec, mode, ignoreVersionMismatch)
}

// InstallRecordedLibrary installs a library even though libraries.json already
// lists it. `sculk install` needs this: every library it handles is, by
// definition, already recorded, so the IsPreinstalled guard in
// InstallLibraryWithMode would make it skip everything.
//
// In "separate" mode a pack that is already on disk is left alone, which keeps
// `sculk install` safe to re-run.
func InstallRecordedLibrary(spec LibrarySpec, mode string, ignoreVersionMismatch bool) error {
	if mode == "" {
		mode = DefaultInstallMode()
	}
	spec.Identifier = lookupIdentifier(spec)

	block := VerifyLibraryIntegrity(spec.Identifier)

	if block.Notice != "" {
		log.Printf("ℹ '%s': %s", block.Identifier, block.Notice)
	}

	resolved, err := Resolve(spec)
	if err != nil {
		return err
	}
	resolved.Block = block
	resolved.Mode = mode

	if mode == create.InstallSeparate {
		if target, _, err := SeparateTargetDir(block); err == nil {
			if entries, err := os.ReadDir(target); err == nil && len(entries) > 0 {
				log.Printf("ℹ '%s' is already present at %s, skipping.", block.Identifier, target)
				return nil
			}
		}
	}

	project := ReadLocalLibrariesJson()

	// Match Game Versions, unless --ignore was given.
	if incompatible, projectGV, libraryGV := gameVersionIncompat(project.GameVersion, resolved.Meta.GameVersion); incompatible {
		if !ignoreVersionMismatch {
			return fmt.Errorf(
				"'%s' targets Minecraft %s but this project is on %s; change game_version in libraries.json, pick another ref ('sculk add %s@/%s'), or pass --ignore to install anyway",
				resolved.Block.Identifier, libraryGV, projectGV, resolved.Block.Identifier, projectGV,
			)
		}
		log.Printf("⚠ Ignoring game version mismatch (%s wants %s, project is %s) because --ignore was given.",
			resolved.Block.Identifier, libraryGV, projectGV)
	}

	if mode == create.InstallMerge {
		log.Printf("📦 Merging '%s' into this project ...", resolved.Block.Identifier)
		if err := MergeIndividualFiles(resolved.Fs, resolved.Root, workingDir()); err != nil {
			return err
		}
	} else {
		target, err := InstallAsSeparatePack(resolved)
		if err != nil {
			return err
		}
		log.Printf("📦 Installed '%s' as a separate pack at %s", resolved.Block.Identifier, target)
	}

	// Record it in this project's libraries.json either way, so `sculk install`
	// can reproduce the setup on another machine.
	return AddToLibrariesJsonRecord(resolved)
}

// lookupIdentifier strips a spec suffix off a raw identifier. Direct URLs are
// returned untouched.
func lookupIdentifier(spec LibrarySpec) string {
	if spec.IsURL {
		return spec.Identifier
	}
	return spec.Identifier
}

// Resolve maps a spec all the way to an in-memory checkout.
func Resolve(spec LibrarySpec) (ResolvedLibrary, error) {
	var out ResolvedLibrary
	out.Spec = spec

	block := VerifyLibraryIntegrity(spec.Identifier)
	out.Block = block

	repoURL := block.Source
	if repoURL == "" {
		return out, fmt.Errorf("no source registered for '%s'", spec.Identifier)
	}
	out.RepoURL = repoURL

	// Ask the registry (or the local project) which game version we should
	// prefer when the caller did not pin one.
	projectGV := ReadLocalLibrariesJson().GameVersion

	ref, hasRef, err := ResolveRef(repoURL, spec, projectGV)
	if err != nil {
		log.Printf("⚠ %v - falling back to the default branch.", err)
	}
	out.Ref = ref
	out.HasRef = hasRef

	// Clone into memory.
	fs := memfs.New()
	cloneOpts := &git.CloneOptions{URL: repoURL, Depth: 1}
	if hasRef {
		cloneOpts.ReferenceName = ref.RefName
		cloneOpts.SingleBranch = true
	}
	if _, err := git.Clone(memory.NewStorage(), fs, cloneOpts); err != nil {
		return out, fmt.Errorf("could not clone %s: %w", repoURL, err)
	}
	out.Fs = fs

	// Monorepo support: descend into the registered subdirectory.
	out.Root = "/"
	if block.Subdir != "" {
		sub := "/" + strings.Trim(block.Subdir, "/")
		if info, err := fs.Stat(sub); err != nil || !info.IsDir() {
			return out, fmt.Errorf("subdirectory '%s' not found in %s", block.Subdir, repoURL)
		}
		out.Root = sub
		log.Printf("📂 Using subdirectory '%s' of %s", block.Subdir, filepath.Base(strings.TrimSuffix(repoURL, ".git")))
	}

	// The library's own libraries.json, synthesized when it does not ship one.
	out.Meta = readLibraryMeta(fs, out.Root, block, ref, hasRef)

	log.Printf("📩 Downloaded %s (%s)", repoURL, describeResolved(out))
	return out, nil
}

// describeResolved renders the version/ref for log output.
func describeResolved(r ResolvedLibrary) string {
	parts := []string{"v" + r.Meta.Version}
	if r.Meta.GameVersion != "" {
		parts = append(parts, "Minecraft "+r.Meta.GameVersion)
	}
	if r.HasRef {
		parts = append(parts, string(r.Ref.Kind)+" "+r.Ref.Name)
	}
	return strings.Join(parts, ", ")
}

// readLibraryMeta loads <root>/libraries.json, falling back to information
// sculk already knows when the library does not ship one.
func readLibraryMeta(fs billy.Filesystem, root string, block libraryBlock, ref RemoteRef, hasRef bool) create.LibrariesDotJson {
	meta := create.LibrariesDotJson{
		Name:        block.Identifier,
		Author:      block.Author,
		Version:     "0.0.0",
		GameVersion: "",
		Libraries:   []create.Library{},
	}

	path := filepath.Join(root, "libraries.json")
	if f, err := fs.Open(path); err == nil {
		defer f.Close()
		data, err := io.ReadAll(f)
		if err == nil {
			var parsed create.LibrariesDotJson
			if err := json.Unmarshal(data, &parsed); err == nil {
				meta = parsed
			} else {
				log.Printf("⚠ '%s' ships an unparsable libraries.json (%v); using derived metadata.", block.Identifier, err)
			}
		}
	}

	// Fill in whatever the library did not declare, preferring the resolved
	// ref over the registry defaults.
	if meta.Name == "" {
		meta.Name = block.Identifier
	}
	if meta.Author == "" {
		meta.Author = block.Author
	}
	if meta.Version == "" || meta.Version == "0.0.0" {
		if hasRef && !ref.Version.IsZero() {
			meta.Version = ref.Version.String()
		} else if block.DefaultVersion != "" {
			meta.Version = block.DefaultVersion
		}
	}
	if meta.GameVersion == "" {
		if hasRef && !ref.GameVersion.IsZero() {
			meta.GameVersion = ref.GameVersion.String()
		} else if block.DefaultGameVersion != "" {
			meta.GameVersion = block.DefaultGameVersion
		}
	}

	return meta
}

func workingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}

// IsPreinstalled reports whether identifier is already recorded in this
// project's libraries.json.
func IsPreinstalled(libraryIdentifier string) bool {
	log.Printf("🔍 Checking in libraries.json ...")

	var librariesFile create.LibrariesDotJson
	filePath := filepath.Join(workingDir(), "libraries.json")
	if _, err := os.Stat(filePath); err != nil {
		log.Printf("⚠ No libraries.json here - run 'sculk init' first.")
		return false
	}

	librariesInstalled := ReadLocalLibrariesJson().Libraries
	for i := range librariesInstalled {
		if strings.EqualFold(librariesInstalled[i].Identifier, libraryIdentifier) {
			log.Printf("🔍 Library '%s' is already installed.", libraryIdentifier)
			return true
		}
	}
	_ = librariesFile
	return false
}

// READS WORKING DIRECTORY'S libraries.json FILE.
func ReadLocalLibrariesJson() create.LibrariesDotJson {
	var libraryJson create.LibrariesDotJson

	filePath := filepath.Join(workingDir(), "libraries.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		// A project that has not been initialized yet still has a sensible
		// empty shape; callers only look at GameVersion and Libraries.
		libraryJson.Libraries = []create.Library{}
		return libraryJson
	}
	if err := json.Unmarshal(data, &libraryJson); err != nil {
		log.Printf("⚠ Could not parse %s: %v", filePath, err)
	}
	if libraryJson.Libraries == nil {
		libraryJson.Libraries = []create.Library{}
	}
	return libraryJson
}

// gameVersionIncompat tests the project's game version against the game
// version the library declares.
//
// The library's game_version is read as a constraint expression, so a library
// may pin an exact version ("26.2") or declare a range (">=26.2",
// "<1.20.1", ">=1.20 <26.0"). An empty game version on either side is treated
// as "unknown" and therefore compatible.
func gameVersionIncompat(projectGameVersion, libraryGameVersion string) (isIncompatible bool, existingProjectVersion string, intendedGameVersion string) {
	if strings.TrimSpace(projectGameVersion) == "" || strings.TrimSpace(libraryGameVersion) == "" {
		return false, projectGameVersion, libraryGameVersion
	}
	if version.ParseConstraint(libraryGameVersion).MatchesString(projectGameVersion) {
		return false, projectGameVersion, libraryGameVersion
	}
	return true, projectGameVersion, libraryGameVersion
}

// dont merge contents of these files from imported libraries.
func avoidFileName(fileName string) bool {
	var BlacklistedFileNames = []string{
		"README.md",
		"LICENSE",
		"pack.mcmeta",
		"libraries.json",
	}
	if slices.Contains(BlacklistedFileNames, fileName) {
		return true
	} else {
		return false
	}
}

func MergeIndividualFiles(fs billy.Filesystem, currentPath string, targetDir string) error {

	files, err := fs.ReadDir(currentPath)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", currentPath, err)
	}

	for _, file := range files {

		// skip LICENSE file
		if avoidFileName(file.Name()) {
			// LOGGER: Ignoring README.md
			log.Printf("⛔ Ignoring %s", file.Name())
			continue
		}

		// store memory path and local path
		memoryPath := filepath.Join(currentPath, file.Name())
		localPath := filepath.Join(targetDir, file.Name())

		if file.IsDir() {
			// if found directory, go inside that directory and recurse.
			if err := MergeIndividualFiles(fs, memoryPath, localPath); err != nil {
				return err
			}

		} else {
			if err := handleFileMerging(fs, memoryPath, localPath); err != nil {
				return err
			}
			log.Printf("🎉 Merged %s", file.Name())
		}

	}

	return nil
}

type FunctionTag struct {
	Replace bool     `json:"replace"`
	Values  []string `json:"values"`
}

func handleFileMerging(fs billy.Filesystem, sourcePath string, destinationPath string) error {

	srcFile, err := fs.Open(sourcePath)
	if err != nil {
		log.Error("⚠ Source File Doesn't Exist.")
		return err
	}
	defer srcFile.Close()

	err = os.MkdirAll(filepath.Dir(destinationPath), 0755)
	if err != nil {
		return err
	}

	// if file already exists, append contents to the top of the file.
	if fileInfo, err := os.Stat(destinationPath); err == nil {

		fileName := strings.Split(fileInfo.Name(), ".")
		fileExtension := ""
		for i := range fileName {

			if i == len(fileName)-1 {
				fileExtension = fileName[i]
			}

		}
		if fileExtension == "json" {

			// LOGGER: Merging to load.json & tick.json

			if fileInfo.Name() == "load.json" || fileInfo.Name() == "tick.json" {
				log.Printf("📩 Merging into %s", fileInfo.Name())

				var sourceTagContent FunctionTag
				var existingTagContent FunctionTag

				existingContent, err := os.ReadFile(destinationPath)
				if err != nil {
					return err
				}

				sourceContent, err := io.ReadAll(srcFile)
				if err != nil {
					return err
				}

				if err := json.Unmarshal(sourceContent, &sourceTagContent); err != nil {
					return err
				}

				if err := json.Unmarshal(existingContent, &existingTagContent); err != nil {
					return err
				}

				existingTagContent.Values = appendTagValues(existingTagContent.Values, sourceTagContent.Values)
				combined, err := json.MarshalIndent(existingTagContent, "", "  ")
				if err != nil {
					return err
				}
				return os.WriteFile(destinationPath, combined, 0644)

			} else {
				return nil
			}
		}

		existingData, err := os.ReadFile(destinationPath)
		if err != nil {
			return err
		}

		newData, err := io.ReadAll(srcFile)
		if err != nil {
			return err
		}

		combined := append(newData, []byte("\n# added by sculk ^^\n")...)
		combined = append(combined, existingData...)

		return os.WriteFile(destinationPath, combined, 0644)
	}

	// if file doesn't exist:
	destinationFile, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, srcFile)
	return err

}

// appendTagValues merges two tag value lists without producing duplicates, so
// that installing (and re-installing) a library does not grow load.json or
// tick.json forever.
func appendTagValues(existing, incoming []string) []string {
	seen := make(map[string]bool, len(existing))
	out := make([]string, 0, len(existing)+len(incoming))
	for _, v := range existing {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	for _, v := range incoming {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// invalidPathChars are replaced when an identifier becomes a directory name.
var invalidPathChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// SanitizeFolderName turns an identifier or URL into a safe directory name.
func SanitizeFolderName(name string) string {
	// For a URL, take the repository name.
	if IsDirectURL(name) {
		trimmed := strings.TrimSuffix(name, ".git")
		if idx := strings.LastIndexAny(trimmed, "/:"); idx >= 0 {
			trimmed = trimmed[idx+1:]
		}
		name = trimmed
	}
	clean := invalidPathChars.ReplaceAllString(name, "-")
	clean = strings.Trim(clean, "-.")
	if clean == "" {
		clean = "sculk-library"
	}
	return clean
}
