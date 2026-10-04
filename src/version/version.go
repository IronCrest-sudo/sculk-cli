// Package version implements the (loose) semantic-version parsing and
// constraint matching used all over sculk-cli.
//
// It deliberately accepts more than strict semver, because Minecraft game
// versions ("26.2", "1.20.1", "26.4-snapshot-1") and library versions
// ("1.0.0") are both routed through the same code paths.
//
// A version is a dot separated list of numeric segments with an optional
// pre-release suffix:
//
//	1.0.0
//	26.2
//	26.4-snapshot-1
//	1.20.1-rc1
//
// A constraint is one or more comparators joined by AND (a comma or
// whitespace) and by OR ("||"):
//
//	1.0.0              exact
//	=1.0.0  ==1.0.0    exact
//	!=1.0.0            not equal
//	<1.20.1  <=1.20.1  greater/lesser
//	>26.2    >=26.2
//	^1.2.3             >=1.2.3 <2.0.0   (caret)
//	~1.2.3             >=1.2.3 <1.3.0   (tilde)
//	1.2.x   1.2.*      >=1.2.0 <1.3.0   (wildcard)
//	>=1.0.0 <2.0.0     AND
//	1.0.0 || 2.0.0     OR
//	*  latest  ""      matches everything
package version

import (
	"strconv"
	"strings"
)

// Version is a parsed version string.
type Version struct {
	// Segments holds the numeric part, e.g. [26, 4] for "26.4-snapshot-1".
	Segments []int
	// Pre is the pre-release suffix without the leading '-', "" when absent.
	Pre string
	// Raw is the original, untouched input.
	Raw string
}

// Sentinel tokens that are accepted anywhere a version is expected and that
// mean "any version".
func isWildcardToken(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "*", "x", "latest", "any", "newest":
		return true
	}
	return false
}

// IsWildcard reports whether s is one of the "matches everything" tokens.
func IsWildcard(s string) bool { return isWildcardToken(s) }

// Parse parses a version string. Unknown trailing junk is kept in Pre so that
// nothing is silently dropped. Parsing never fails hard: an unparsable
// leading segment simply yields an empty Segments slice.
func Parse(s string) Version {
	raw := s
	s = strings.TrimSpace(s)

	// Strip a leading "v" ("v1.0.0" == "1.0.0").
	if strings.HasPrefix(s, "v") || strings.HasPrefix(s, "V") {
		s = s[1:]
	}

	var v Version
	v.Raw = raw

	// Split off the pre-release part on the first '-' that is not part of a
	// numeric segment ("26.4-snapshot-1").
	core := s
	if i := strings.Index(core, "-"); i >= 0 {
		v.Pre = core[i+1:]
		core = core[:i]
	}
	// '+' build metadata is treated like a pre-release suffix for ordering.
	if i := strings.Index(core, "+"); i >= 0 {
		if v.Pre == "" {
			v.Pre = core[i+1:]
		}
		core = core[:i]
	}

	if core != "" {
		for _, part := range strings.Split(core, ".") {
			part = strings.TrimSpace(part)
			// "x"/"*" wildcard inside a version means "everything after here".
			if part == "x" || part == "X" || part == "*" {
				break
			}
			n, err := strconv.Atoi(part)
			if err != nil {
				// Non numeric segment: keep the remainder as pre-release.
				if v.Pre == "" {
					v.Pre = strings.Join(strings.Split(s, ".")[len(v.Segments):], ".")
				}
				break
			}
			v.Segments = append(v.Segments, n)
		}
	}

	return v
}

// String renders the version back to a comparable string.
func (v Version) String() string {
	if len(v.Segments) == 0 && v.Pre == "" {
		return v.Raw
	}
	parts := make([]string, len(v.Segments))
	for i, n := range v.Segments {
		parts[i] = strconv.Itoa(n)
	}
	out := strings.Join(parts, ".")
	if v.Pre != "" {
		out += "-" + v.Pre
	}
	return out
}

// IsZero reports whether nothing numeric could be parsed out of the input.
func (v Version) IsZero() bool { return len(v.Segments) == 0 && v.Pre == "" }

// Compare returns -1/0/1 when a is lower/equal/higher than b.
//
// Missing trailing segments count as zero, so "26.2" == "26.2.0".
// A pre-release sorts below its release: "26.4-snapshot-1" < "26.4".
func (a Version) Compare(b Version) int {
	n := len(a.Segments)
	if len(b.Segments) > n {
		n = len(b.Segments)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(a.Segments) {
			x = a.Segments[i]
		}
		if i < len(b.Segments) {
			y = b.Segments[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}

	switch {
	case a.Pre == "" && b.Pre == "":
		return 0
	case a.Pre == "":
		return 1 // release > pre-release
	case b.Pre == "":
		return -1
	default:
		return strings.Compare(a.Pre, b.Pre)
	}
}

// Less/Equal/Greater are convenience wrappers around Compare.
func (a Version) Less(b Version) bool    { return a.Compare(b) < 0 }
func (a Version) Equal(b Version) bool   { return a.Compare(b) == 0 }
func (a Version) Greater(b Version) bool { return a.Compare(b) > 0 }

// CompareStrings parses both inputs and compares them.
func CompareStrings(a, b string) int { return Parse(a).Compare(Parse(b)) }
