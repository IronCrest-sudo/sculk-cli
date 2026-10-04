package add

import (
	"fmt"
	"strings"

	"sculk-cli/src/version"
)

// LibrarySpec is a parsed `sculk add` argument.
//
// The grammar accepted on the command line is:
//
//	<identifier>                          latest for the current GameVer
//	<identifier>@latest                   same as above
//	<identifier>@<version>                that version, current GameVer
//	<identifier>@<version>/<gameVer>      that version of that GameVer
//	<identifier>@/<gameVer>               latest version for that GameVer
//	<url>                                 a direct git link, never a spec
//
// <version> and <gameVer> are full constraint expressions, so all of these are
// legal too:
//
//	id-system@^1.0.0
//	id-system@">=1.0.0 <2.0.0"
//	id-system@1.20.1/26.2
//	id-system@<1.20.1
type LibrarySpec struct {
	// Identifier is the registry key, or the raw URL for direct installs.
	Identifier string
	// Version is the raw version constraint text, "" when omitted.
	Version string
	// GameVersion is the raw game-version constraint text, "" when omitted.
	GameVersion string
	// IsURL is true when the argument was a direct git link.
	IsURL bool
	// Raw is the untouched argument.
	Raw string
}

// LatestKeyword is the reserved version token meaning "newest available".
const LatestKeyword = "latest"

// IsDirectURL reports whether arg points at a git host rather than a registry
// identifier. This has to run before spec parsing because URLs contain both
// '/' and, occasionally, '@'.
func IsDirectURL(arg string) bool {
	a := strings.TrimSpace(arg)
	if a == "" {
		return false
	}
	if strings.Contains(a, "://") {
		return true
	}
	lower := strings.ToLower(a)
	for _, host := range []string{"github.com/", "codeberg.org/", "gitlab.com/", "git.sr.ht/", "bitbucket.org/"} {
		if strings.HasPrefix(lower, host) {
			return true
		}
	}
	// scp-like ssh form: git@github.com:owner/repo.git
	if strings.HasPrefix(lower, "git@") && strings.Contains(a, ":") {
		return true
	}
	return false
}

// ParseSpec splits an `sculk add` argument into identifier, version and game
// version. A direct URL is returned untouched with IsURL set.
func ParseSpec(arg string) (LibrarySpec, error) {
	raw := arg
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return LibrarySpec{}, fmt.Errorf("empty library identifier")
	}

	if IsDirectURL(arg) {
		// A browser link to a registered library ("…/tree/main/<subdir>") is
		// just another spelling of its registry identifier.
		if id, ok := RegistryIdentifierForURL(arg); ok {
			return LibrarySpec{Identifier: id, Raw: raw}, nil
		}
		return LibrarySpec{Identifier: arg, IsURL: true, Raw: raw}, nil
	}

	spec := LibrarySpec{Raw: raw, Identifier: arg}

	// Split on the FIRST '@'. Everything after it is the version part.
	at := strings.Index(arg, "@")
	if at >= 0 {
		spec.Identifier = strings.TrimSpace(arg[:at])
		versionPart := strings.TrimSpace(arg[at+1:])

		if versionPart != "" {
			// The game version may be given as a second path segment:
			//   1.0.0/26.2   or   /26.2
			if slash := strings.Index(versionPart, "/"); slash >= 0 {
				spec.Version = strings.TrimSpace(versionPart[:slash])
				spec.GameVersion = strings.TrimSpace(versionPart[slash+1:])
			} else {
				spec.Version = versionPart
			}
		}
	}

	if spec.Identifier == "" {
		return LibrarySpec{}, fmt.Errorf("missing library identifier in %q", raw)
	}

	return spec, nil
}

// VersionConstraint returns the parsed version constraint. An omitted version,
// the "latest" keyword and "*" all collapse to a match-everything constraint.
func (s LibrarySpec) VersionConstraint() version.Constraint {
	if s.IsLatest() {
		return version.Constraint{Wildcard: true, Raw: s.Version}
	}
	return version.ParseConstraint(s.Version)
}

// GameVersionConstraint returns the parsed game-version constraint.
func (s LibrarySpec) GameVersionConstraint() version.Constraint {
	return version.ParseConstraint(s.GameVersion)
}

// IsLatest reports whether the caller asked for "the newest thing" instead of
// a specific version.
func (s LibrarySpec) IsLatest() bool {
	return version.IsWildcard(s.Version)
}

// WantsSpecificGameVersion reports whether the caller pinned a game version.
func (s LibrarySpec) WantsSpecificGameVersion() bool {
	return !version.IsWildcard(s.GameVersion)
}

// String renders the spec back into its canonical CLI form, which is handy
// for log output ("id-system@1.0.0/26.2").
func (s LibrarySpec) String() string {
	if s.IsURL {
		return s.Identifier
	}
	hasVer := !version.IsWildcard(s.Version)
	hasGV := !version.IsWildcard(s.GameVersion)
	switch {
	case !hasVer && !hasGV:
		return s.Identifier
	case hasVer && hasGV:
		return s.Identifier + "@" + s.Version + "/" + s.GameVersion
	case hasVer:
		return s.Identifier + "@" + s.Version
	default:
		return s.Identifier + "@/" + s.GameVersion
	}
}
