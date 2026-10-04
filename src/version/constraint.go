package version

import (
	"strings"
)

// Comparator is a single "op version" test.
type Comparator struct {
	Op      string // "=", "!=", "<", "<=", ">", ">=", "^", "~", "x"
	Version Version
}

// ConstraintSet is one AND-group of comparators.
type ConstraintSet []Comparator

// Constraint is a parsed constraint expression: an OR of AND-groups.
type Constraint struct {
	Sets []ConstraintSet
	// Wildcard is true for "", "*", "x", "latest", "any" - matches everything.
	Wildcard bool
	Raw      string
}

// operators, longest first so that "<=" is not read as "<" + "="
var operators = []string{"==", "!=", "<=", ">=", "=", "<", ">", "^", "~"}

// Matches reports whether v satisfies this comparator.
func (c Comparator) Matches(v Version) bool {
	switch c.Op {
	case "=", "==":
		return v.Equal(c.Version)
	case "!=":
		return !v.Equal(c.Version)
	case "<":
		return v.Less(c.Version)
	case "<=":
		return v.Compare(c.Version) <= 0
	case ">":
		return v.Greater(c.Version)
	case ">=":
		return v.Compare(c.Version) >= 0
	case "^":
		// >=X.Y.Z <(next significant)0.0
		lo := c.Version
		hi := bumpCaret(c.Version)
		return v.Compare(lo) >= 0 && v.Less(hi)
	case "~":
		lo := c.Version
		hi := bumpTilde(c.Version)
		return v.Compare(lo) >= 0 && v.Less(hi)
	case "x":
		// Wildcard form "1.2.x": the wildcard position itself is what bumps.
		lo := c.Version
		hi := bumpWildcard(c.Version)
		return v.Compare(lo) >= 0 && v.Less(hi)
	}
	// Unknown operator: never match rather than silently accept.
	return false
}

// bumpCaret returns the exclusive upper bound for "^X.Y.Z".
// The left-most non-zero segment is the one that gets bumped:
//
//	^1.2.3 -> 2.0.0
//	^0.2.3 -> 0.3.0
//	^0.0.3 -> 0.0.4
func bumpCaret(v Version) Version {
	segs := make([]int, len(v.Segments))
	copy(segs, v.Segments)
	if len(segs) == 0 {
		return Version{Segments: []int{1}}
	}
	for i := range segs {
		if segs[i] != 0 {
			segs[i]++
			for j := i + 1; j < len(segs); j++ {
				segs[j] = 0
			}
			return Version{Segments: segs}
		}
	}
	// all zeros -> ^0.0.0 matches only 0.0.x
	last := len(segs) - 1
	segs[last]++
	return Version{Segments: segs}
}

// bumpTilde returns the exclusive upper bound for "~X.Y.Z": >=X.Y.Z <X.(Y+1).0
func bumpTilde(v Version) Version {
	segs := make([]int, len(v.Segments))
	copy(segs, v.Segments)
	if len(segs) == 0 {
		return Version{Segments: []int{1}}
	}
	if len(segs) == 1 {
		// ~1 means >=1.0.0 <2.0.0
		segs[0]++
		return Version{Segments: segs}
	}
	segs[1]++
	for j := 2; j < len(segs); j++ {
		segs[j] = 0
	}
	return Version{Segments: segs[:2]}
}

// bumpWildcard returns the exclusive upper bound for the wildcard form
// "A.B.x": the last *written* segment is the one that bumps, because the
// wildcard itself is not part of Version.Segments.
//
//	1.2.x -> 1.3
//	1.x   -> 2
//	26.4.x -> 26.5
func bumpWildcard(v Version) Version {
	segs := make([]int, len(v.Segments))
	copy(segs, v.Segments)
	if len(segs) == 0 {
		return Version{Segments: []int{1}}
	}
	segs[len(segs)-1]++
	return Version{Segments: segs}
}

// ParseConstraint parses a constraint expression. Unparsable input yields a
// wildcard constraint (match everything) so that a typo in libraries.json
// never hard-blocks an install - callers that need strictness should check
// Constraint.Wildcard.
func ParseConstraint(expr string) Constraint {
	raw := expr
	expr = strings.TrimSpace(expr)

	if isWildcardToken(expr) {
		return Constraint{Wildcard: true, Raw: raw}
	}

	var sets []ConstraintSet
	for _, orPart := range strings.Split(expr, "||") {
		orPart = strings.TrimSpace(orPart)
		if orPart == "" {
			continue
		}
		if isWildcardToken(orPart) {
			return Constraint{Wildcard: true, Raw: raw}
		}

		// AND-group: split on commas and whitespace, but keep an operator
		// glued to its version when there is no separator (">=1.0.0<2.0.0").
		tokens := tokenize(orPart)
		if len(tokens) == 0 {
			continue
		}

		set := make(ConstraintSet, 0, len(tokens))
		for _, tok := range tokens {
			c, ok := parseComparator(tok)
			if !ok {
				continue
			}
			set = append(set, c)
		}
		if len(set) > 0 {
			sets = append(sets, set)
		}
	}

	if len(sets) == 0 {
		return Constraint{Wildcard: true, Raw: raw}
	}
	return Constraint{Sets: sets, Raw: raw}
}

// tokenize splits an AND-group into comparator tokens. It keeps a leading
// operator attached to the version that follows it.
func tokenize(s string) []string {
	var out []string
	var cur strings.Builder

	flush := func() {
		if strings.TrimSpace(cur.String()) != "" {
			out = append(out, strings.TrimSpace(cur.String()))
		}
		cur.Reset()
	}

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if r == ',' || r == ' ' || r == '\t' {
			flush()
			continue
		}

		if isOperatorStart(r) {
			// A new comparator starts here: flush what we have, then take the
			// full operator (<=, >=, ==, !=) in one go.
			flush()
			op := string(r)
			if i+1 < len(runes) && (runes[i+1] == '=') {
				op += "="
				i++
			}
			cur.WriteString(op)
			continue
		}

		cur.WriteRune(r)
	}
	flush()
	return out
}

func isOperatorStart(r rune) bool {
	switch r {
	case '<', '>', '=', '!', '^', '~':
		return true
	}
	return false
}

// parseComparator turns a single token such as ">=26.2" into a Comparator.
func parseComparator(tok string) (Comparator, bool) {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return Comparator{}, false
	}

	op := "="
	rest := tok
	for _, candidate := range operators {
		if strings.HasPrefix(tok, candidate) {
			op = candidate
			rest = strings.TrimSpace(strings.TrimPrefix(tok, candidate))
			break
		}
	}

	if rest == "" {
		return Comparator{}, false
	}

	// Wildcards inside a comparator ("1.2.x", "*") expand to a range.
	if isWildcardToken(rest) {
		return Comparator{Op: ">=", Version: Version{Segments: []int{0}}}, true
	}

	v := Parse(rest)

	// Reject anything without a numeric part ("not-a-version", "snapshot")
	// instead of silently turning it into a nonsense comparator.
	if len(v.Segments) == 0 {
		return Comparator{}, false
	}

	// "1.2.x" / "1.2.*" -> >=1.2.0 <1.3.0 (the wildcard segment bumps)
	if op == "=" && hasTrailingWildcard(rest) {
		return Comparator{Op: "x", Version: v}, true
	}

	if op == "==" {
		op = "="
	}
	return Comparator{Op: op, Version: v}, true
}

// hasTrailingWildcard reports whether the last dot-separated segment of a
// version expression is a wildcard, e.g. "1.2.x", "1.*", "26.x".
func hasTrailingWildcard(expr string) bool {
	parts := strings.Split(expr, ".")
	last := strings.TrimSpace(parts[len(parts)-1])
	switch last {
	case "x", "X", "*":
		return true
	}
	return false
}

// Matches reports whether v satisfies the constraint.
// An empty set of comparators (or a wildcard) matches everything.
func (c Constraint) Matches(v Version) bool {
	if c.Wildcard {
		return true
	}
	for _, set := range c.Sets {
		ok := true
		for _, cmp := range set {
			if !cmp.Matches(v) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// MatchesString parses candidate and tests it against the constraint.
func (c Constraint) MatchesString(candidate string) bool {
	if c.Wildcard {
		return true
	}
	return c.Matches(Parse(candidate))
}

// MatchesAny reports whether any of the candidates satisfies the constraint.
func (c Constraint) MatchesAny(candidates []string) bool {
	if c.Wildcard {
		return len(candidates) > 0
	}
	for _, cand := range candidates {
		if c.MatchesString(cand) {
			return true
		}
	}
	return false
}

// MatchesString is a package level convenience for one-shot checks such as
// `version.Satisfies("26.4", "<26.2 || >=26.4")`.
func Satisfies(v, expr string) bool {
	return ParseConstraint(expr).MatchesString(v)
}

// String renders the constraint back to its source expression.
func (c Constraint) String() string {
	if c.Wildcard {
		return "*"
	}
	return c.Raw
}
