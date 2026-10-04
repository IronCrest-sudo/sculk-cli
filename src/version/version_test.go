package version

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		segs []int
		pre  string
	}{
		{"1.0.0", []int{1, 0, 0}, ""},
		{"26.2", []int{26, 2}, ""},
		{"v1.2.3", []int{1, 2, 3}, ""},
		{"26.4-snapshot-1", []int{26, 4}, "snapshot-1"},
		{"1.20.1-rc1", []int{1, 20, 1}, "rc1"},
		{"  1.0  ", []int{1, 0}, ""},
	}

	for _, c := range cases {
		got := Parse(c.in)
		if len(got.Segments) != len(c.segs) {
			t.Fatalf("Parse(%q) segments = %v, want %v", c.in, got.Segments, c.segs)
		}
		for i := range c.segs {
			if got.Segments[i] != c.segs[i] {
				t.Fatalf("Parse(%q) segments = %v, want %v", c.in, got.Segments, c.segs)
			}
		}
		if got.Pre != c.pre {
			t.Fatalf("Parse(%q).Pre = %q, want %q", c.in, got.Pre, c.pre)
		}
	}
}

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		// trailing zeros are insignificant
		{"26.2", "26.2.0", 0},
		{"1.0", "1.0.0.0", 0},
		// game versions
		{"26.2", "26.4", -1},
		{"1.20.1", "26.2", -1},
		// pre-release sorts below release
		{"26.4-snapshot-1", "26.4", -1},
		{"26.4", "26.4-snapshot-1", 1},
		{"1.0.0-alpha", "1.0.0-beta", -1},
		// more segments
		{"1.2.3", "1.2", 1},
	}

	for _, c := range cases {
		got := CompareStrings(c.a, c.b)
		if got != c.want {
			t.Errorf("CompareStrings(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestSatisfies(t *testing.T) {
	cases := []struct {
		v, expr string
		want    bool
	}{
		// exact
		{"1.0.0", "1.0.0", true},
		{"1.0.1", "1.0.0", false},
		{"1.0.0", "=1.0.0", true},
		{"1.0.0", "==1.0.0", true},
		{"1.0.1", "!=1.0.0", true},
		{"1.0.0", "!=1.0.0", false},

		// comparison - the README examples
		{"1.20.0", "<1.20.1", true},
		{"1.20.1", "<1.20.1", false},
		{"1.20.2", ">1.20.1", true},
		{"26.3", ">26.2", true},
		{"26.2", ">26.2", false},
		{"26.2", ">=26.2", true},
		{"26.2", "<=26.2", true},

		// caret
		{"1.2.3", "^1.2.3", true},
		{"1.9.9", "^1.2.3", true},
		{"2.0.0", "^1.2.3", false},
		{"1.2.2", "^1.2.3", false},
		{"0.2.5", "^0.2.3", true},
		{"0.3.0", "^0.2.3", false},

		// tilde
		{"1.2.9", "~1.2.3", true},
		{"1.3.0", "~1.2.3", false},

		// wildcard
		{"1.2.7", "1.2.x", true},
		{"1.3.0", "1.2.x", false},
		{"5.0.0", "*", true},
		{"5.0.0", "latest", true},
		{"5.0.0", "", true},

		// AND
		{"1.5.0", ">=1.0.0 <2.0.0", true},
		{"2.5.0", ">=1.0.0 <2.0.0", false},
		{"1.5.0", ">=1.0.0,<2.0.0", true},
		{"1.5.0", ">=1.0.0<2.0.0", true},

		// OR
		{"1.0.0", "1.0.0 || 2.0.0", true},
		{"2.0.0", "1.0.0 || 2.0.0", true},
		{"3.0.0", "1.0.0 || 2.0.0", false},
		{"26.4", "<26.2 || >=26.4", true},
		{"26.3", "<26.2 || >=26.4", false},

		// game version style
		{"26.2", "26.2", true},
		{"26.4-snapshot-1", "26.4-snapshot-1", true},
		{"26.4-snapshot-1", "^26.4", false}, // pre-release sorts below 26.4
		{"26.4", "^26.4", true},
		// ^26.4 is >=26.4 <27.0.0, so 26.5 is inside the range
		{"26.5", "^26.4", true},
		{"27.0", "^26.4", false},
	}

	for _, c := range cases {
		got := Satisfies(c.v, c.expr)
		if got != c.want {
			t.Errorf("Satisfies(%q, %q) = %v, want %v", c.v, c.expr, got, c.want)
		}
	}
}

func TestParseConstraintWildcardOnGarbage(t *testing.T) {
	// Garbage must degrade to "matches everything" instead of panicking or
	// blocking every install.
	c := ParseConstraint("not-a-version")
	if !c.Wildcard {
		t.Errorf("ParseConstraint(garbage).Wildcard = false, want true")
	}
	if !c.MatchesString("1.0.0") {
		t.Errorf("garbage constraint should match anything")
	}
}

func TestBumpCaretAndTilde(t *testing.T) {
	if got := bumpCaret(Parse("1.2.3")).String(); got != "2.0.0" {
		t.Errorf("bumpCaret(1.2.3) = %s, want 2.0.0", got)
	}
	if got := bumpCaret(Parse("0.2.3")).String(); got != "0.3.0" {
		t.Errorf("bumpCaret(0.2.3) = %s, want 0.3.0", got)
	}
	if got := bumpCaret(Parse("0.0.3")).String(); got != "0.0.4" {
		t.Errorf("bumpCaret(0.0.3) = %s, want 0.0.4", got)
	}
	if got := bumpTilde(Parse("1.2.3")).String(); got != "1.3" {
		t.Errorf("bumpTilde(1.2.3) = %s, want 1.3", got)
	}
	if got := bumpTilde(Parse("1")).String(); got != "2" {
		t.Errorf("bumpTilde(1) = %s, want 2", got)
	}
	if got := bumpWildcard(Parse("1.2.x")).String(); got != "1.3" {
		t.Errorf("bumpWildcard(1.2.x) = %s, want 1.3", got)
	}
	if got := bumpWildcard(Parse("1.x")).String(); got != "2" {
		t.Errorf("bumpWildcard(1.x) = %s, want 2", got)
	}
}

func TestHasTrailingWildcard(t *testing.T) {
	for in, want := range map[string]bool{
		"1.2.x":           true,
		"1.*":             true,
		"26.x":            true,
		"1.2.3":           false,
		"1.0.0+exp":       false, // 'x' inside build metadata is not a wildcard
		"26.4-snapshot-1": false,
	} {
		if got := hasTrailingWildcard(in); got != want {
			t.Errorf("hasTrailingWildcard(%q) = %v, want %v", in, got, want)
		}
	}
}
