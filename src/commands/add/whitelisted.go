// Package-level library registry.
//
// sculk has no central package host: an "identifier" is just a friendly alias
// for a git repository (plus, optionally, a subdirectory inside it). This file
// is the source of truth for those aliases and it is also what the website's
// library listing is generated from, so keep both in sync.
package add

import (
	"sort"
	"strings"

	"sculk-cli/src/commands/initProject/create"
)

// PackKind tells a datapack from a resourcepack. It only matters when a
// library is installed with doMerge=false, where sculk has to lay down a
// correct pack.mcmeta for the library to be loadable on its own.
type PackKind string

const (
	KindDatapack     PackKind = "datapack"
	KindResourcepack PackKind = "resourcepack"
)

// libraryBlock is one entry of the registry.
type libraryBlock struct {
	Identifier string
	Source     string
	// Subdir points at a folder inside Source when the library is not the
	// whole repository (monorepos). Empty means "the repo root".
	Subdir string
	// Kind defaults to KindDatapack.
	Kind PackKind
	// Description is a one-liner for `sculk list` and the website.
	Description string
	// Author is shown next to the identifier on the website.
	Author string
	// Homepage is an optional link to the project page.
	Homepage string
	// Notice carries a warning that should reach the user before install,
	// e.g. "archived upstream".
	Notice string
	// DefaultVersion / DefaultGameVersion are used when the library publishes
	// neither a libraries.json nor versioned refs, so that sculk still knows
	// what it is looking at.
	DefaultVersion     string
	DefaultGameVersion string
}

func (b libraryBlock) packKind() PackKind {
	if b.Kind == "" {
		return KindDatapack
	}
	return b.Kind
}

// KindOrDefault is the exported accessor used by `sculk list` and the website
// generator.
func (b libraryBlock) KindOrDefault() PackKind { return b.packKind() }

// approvedLibraries is the registry itself.
//
// Keys must be lower-case, unique and valid Minecraft namespace characters
// where possible, because they double as the identifier written into
// libraries.json.
var approvedLibraries = map[string]libraryBlock{
	"id-system": {
		Identifier:  "id-system",
		Source:      "https://github.com/officialbarden/id-system",
		Description: "Entity ID system - hands out stable scoreboard ids.",
		Author:      "officialbarden",
	},
	"uuid": {
		Identifier:  "uuid",
		Source:      "https://github.com/CJDevZ/UUID-Hex",
		Description: "High-performance UUID <-> hex string conversion.",
		Author:      "CJDev",
	},
	"astar": {
		Identifier:  "astar",
		Source:      "https://github.com/CJDevZ/A-Star-Pathfinding",
		Description: "A* pathfinding as a Minecraft datapack.",
		Author:      "CJDev",
	},
	"playermotion": {
		Identifier:  "playermotion",
		Source:      "https://github.com/MulverineX/player_motion",
		Description: "Enchantment-effect based player motion.",
		Author:      "MulverineX",
	},
	"titlewriter": {
		Identifier:  "titlewriter",
		Source:      "https://github.com/officialbarden/titlewriter",
		Description: "Animate title text with a per-character delay and sound.",
		Author:      "officialbarden",
	},
	"speclib": {
		Identifier:  "speclib",
		Source:      "https://github.com/officialbarden/speclib",
		Description: "Spectate entities with a free camera.",
		Author:      "officialbarden",
	},
	"reef": {
		Identifier:  "reef",
		Source:      "https://github.com/Trioplane/reef",
		Description: "Build presentations entirely in vanilla Minecraft.",
		Author:      "Trioplane",
	},
	"hitmatch": {
		Identifier:  "hitmatch",
		Source:      "https://github.com/picarrow/hit-match",
		Description: "Select entities that exchanged damage with players.",
		Author:      "picarrow",
	},
	"stringlib": {
		Identifier:  "stringlib",
		Source:      "https://github.com/CMDred/StringLib",
		Description: "String manipulation utilities for datapacks.",
		Author:      "CMDred",
	},

	// macroEngine lives in a monorepo, so it is registered with a Subdir
	// instead of a dedicated repository. The pack is archived upstream, so
	// sculk installs it with its archive notice intact and tells the user
	// about the load gate it ships with.
	"macroengine": {
		Identifier:  "macroengine",
		Source:      "https://github.com/IronCrest-sudo/core.git",
		Subdir:      "archived/macroEngine-Datapack-v26.4",
		Kind:        KindDatapack,
		Description: "Macro/module framework: string, cooldown, multi-cmd, math, nbt, geo, hooks, permissions, uuid cache and an input system.",
		Author:      "vortacraftmc",
		Homepage:    "https://github.com/IronCrest-sudo/core/tree/main/archived/macroEngine-Datapack-v26.4",
		Notice:      "archived upstream (read-only, no further fixes). Ships a load gate: the pack stays inert until an operator runs `/function macroengine:gate/v26_4/confirm {format:122}`. Best installed with `sculk config doMerge false`.",

		// The pack ships no libraries.json, so sculk records what the upstream
		// registry (archived/archive.json) says about it.
		DefaultVersion:     "26.4-snapshot-1",
		DefaultGameVersion: "26.4-snapshot-1",
	},
	"macroengine-rp": {
		Identifier:  "macroengine-rp",
		Source:      "https://github.com/IronCrest-sudo/core.git",
		Subdir:      "packs/macroEngine-Resourcepack-v26.4",
		Kind:        KindResourcepack,
		Description: "Companion resource pack for macroEngine (i18n text, sounds, trim assets).",
		Author:      "vortacraftmc",
		Homepage:    "https://github.com/IronCrest-sudo/core/tree/main/packs/macroEngine-Resourcepack-v26.4",
		Notice:      "companion of 'macroengine'; required for engine text, sounds and armor-trim assets.",

		DefaultVersion:     "26.4-snapshot-1",
		DefaultGameVersion: "26.4-snapshot-1",
	},
}

// VerifyLibraryIntegrity maps an identifier to its registry entry.
//
// Unknown identifiers are not an error: they are treated as a direct link so
// that `sculk add https://github.com/owner/repo` keeps working. The returned
// block then carries the identifier as both Identifier and Source.
func VerifyLibraryIntegrity(identifier string) libraryBlock {
	// Identifiers may arrive with an @version suffix from a raw call site;
	// the spec parser normally strips it, but be forgiving here too.
	bare := identifier
	if at := strings.Index(bare, "@"); at > 0 && !IsDirectURL(bare) {
		bare = bare[:at]
	}

	if block, ok := approvedLibraries[strings.ToLower(strings.TrimSpace(bare))]; ok {
		return block
	}

	return libraryBlock{
		Identifier: identifier,
		Source:     identifier,
	}
}

// IsApproved reports whether identifier is in the registry.
func IsApproved(identifier string) bool {
	_, ok := approvedLibraries[strings.ToLower(strings.TrimSpace(identifier))]
	return ok
}

// ListApprovedLibraries returns every registry entry, sorted by identifier.
// `sculk list` and the website both render this.
func ListApprovedLibraries() []libraryBlock {
	out := make([]libraryBlock, 0, len(approvedLibraries))
	for _, block := range approvedLibraries {
		out = append(out, block)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Identifier < out[j].Identifier })
	return out
}

// BuildImportedLibraryMetadata fetches the library and returns the record that
// goes into the local libraries.json.
func BuildImportedLibraryMetadata(libraryBlock libraryBlock) create.Library {
	resolved, err := Resolve(LibrarySpec{Identifier: libraryBlock.Identifier})
	if err != nil {
		panic(err)
	}
	return libraryRecord(resolved)
}

// libraryRecord turns a resolved library into its libraries.json entry.
func libraryRecord(r ResolvedLibrary) create.Library {
	return create.Library{
		Identifier:  r.Block.Identifier,
		Source:      r.RepoURL,
		Version:     r.Meta.Version,
		GameVersion: r.Meta.GameVersion,
		Install:     r.InstallMode(),
		Ref:         r.RefLabel(),
	}
}
