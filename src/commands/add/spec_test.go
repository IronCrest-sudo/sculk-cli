package add

import (
	"testing"

	"sculk-cli/src/version"
)

func TestParseSpec(t *testing.T) {
	cases := []struct {
		in      string
		id      string
		ver     string
		gameVer string
		isURL   bool
	}{
		// plain identifier
		{"id-system", "id-system", "", "", false},

		// README examples
		{"id-system@1.0.0", "id-system", "1.0.0", "", false},
		{"id-system@1.0.0/26.2", "id-system", "1.0.0", "26.2", false},
		{"id-system@/26.2", "id-system", "", "26.2", false},
		{"id-system@latest", "id-system", "latest", "", false},

		// constraint expressions stay intact
		{"id-system@<1.20.1", "id-system", "<1.20.1", "", false},
		{"id-system@>26.2", "id-system", ">26.2", "", false},
		{"id-system@^1.0.0/26.2", "id-system", "^1.0.0", "26.2", false},
		{"id-system@1.2.x", "id-system", "1.2.x", "", false},
		{"id-system@>=1.0.0,<2.0.0", "id-system", ">=1.0.0,<2.0.0", "", false},
		{"id-system@1.0.0||2.0.0", "id-system", "1.0.0||2.0.0", "", false},

		// snapshot style game versions
		{"macroengine@26.4-snapshot-1", "macroengine", "26.4-snapshot-1", "", false},
		{"macroengine@v1.0.0/26.4-snapshot-1", "macroengine", "v1.0.0", "26.4-snapshot-1", false},

		// whitespace tolerated
		{"  id-system@1.0.0 ", "id-system", "1.0.0", "", false},

		// trailing @ with nothing after it
		{"id-system@", "id-system", "", "", false},

		// direct URLs are never specs, even though they contain '/' and '@'
		{"https://github.com/officialbarden/id-system", "https://github.com/officialbarden/id-system", "", "", true},
		{"http://github.com/officialbarden/id-system", "http://github.com/officialbarden/id-system", "", "", true},
		{"github.com/officialbarden/id-system", "github.com/officialbarden/id-system", "", "", true},
		{"https://github.com/a/b/tree/1.0.0", "https://github.com/a/b/tree/1.0.0", "", "", true},
		{"git@github.com:owner/repo.git", "git@github.com:owner/repo.git", "", "", true},
		{"https://codeberg.org/a/b", "https://codeberg.org/a/b", "", "", true},
	}

	for _, c := range cases {
		got, err := ParseSpec(c.in)
		if err != nil {
			t.Fatalf("ParseSpec(%q) error: %v", c.in, err)
		}
		if got.Identifier != c.id {
			t.Errorf("ParseSpec(%q).Identifier = %q, want %q", c.in, got.Identifier, c.id)
		}
		if got.Version != c.ver {
			t.Errorf("ParseSpec(%q).Version = %q, want %q", c.in, got.Version, c.ver)
		}
		if got.GameVersion != c.gameVer {
			t.Errorf("ParseSpec(%q).GameVersion = %q, want %q", c.in, got.GameVersion, c.gameVer)
		}
		if got.IsURL != c.isURL {
			t.Errorf("ParseSpec(%q).IsURL = %v, want %v", c.in, got.IsURL, c.isURL)
		}
	}
}

func TestParseSpecErrors(t *testing.T) {
	for _, in := range []string{"", "   ", "@1.0.0"} {
		if _, err := ParseSpec(in); err == nil {
			t.Errorf("ParseSpec(%q) should fail", in)
		}
	}
}

func TestSpecString(t *testing.T) {
	cases := map[string]string{
		"id-system":            "id-system",
		"id-system@latest":     "id-system",
		"id-system@*":          "id-system",
		"id-system@1.0.0":      "id-system@1.0.0",
		"id-system@1.0.0/26.2": "id-system@1.0.0/26.2",
		"id-system@/26.2":      "id-system@/26.2",
	}
	for in, want := range cases {
		spec, err := ParseSpec(in)
		if err != nil {
			t.Fatal(err)
		}
		if got := spec.String(); got != want {
			t.Errorf("ParseSpec(%q).String() = %q, want %q", in, got, want)
		}
	}
}

func TestParseRefName(t *testing.T) {
	cases := []struct {
		in      string
		ok      bool
		ver     string
		gameVer string
	}{
		{"1.0.0", true, "1.0.0", ""},
		{"v1.0.0", true, "1.0.0", ""},
		{"1.0.0/26.2", true, "1.0.0", "26.2"},
		{"v1.0.0/26.2", true, "1.0.0", "26.2"},
		{"26.4-snapshot-1", true, "26.4-snapshot-1", ""},
		{"2.0/26.4-snapshot-1", true, "2.0", "26.4-snapshot-1"},
		// non-version refs must be ignored
		{"main", false, "", ""},
		{"master", false, "", ""},
		{"develop", false, "", ""},
		{"HEAD", false, "", ""},
		{"latest", false, "", ""},
		{"feature-branch", false, "", ""},
		{"", false, "", ""},
	}

	for _, c := range cases {
		got, ok := parseRefName(c.in)
		if ok != c.ok {
			t.Errorf("parseRefName(%q) ok = %v, want %v", c.in, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if got.Version.String() != c.ver {
			t.Errorf("parseRefName(%q).Version = %q, want %q", c.in, got.Version.String(), c.ver)
		}
		if got.GameVersion.String() != c.gameVer {
			t.Errorf("parseRefName(%q).GameVersion = %q, want %q", c.in, got.GameVersion.String(), c.gameVer)
		}
	}
}

func TestFindRefPrefersBranchOverTag(t *testing.T) {
	refs := []RemoteRef{}
	b, _ := parseRefName("1.0.0")
	b.Kind = RefTag
	b.Name = "1.0.0"
	tag, _ := parseRefName("1.0.0")
	tag.Kind = RefBranch
	tag.Name = "1.0.0"
	refs = append(refs, b, tag)

	got, ok := findRef(refs, "1.0.0")
	if !ok {
		t.Fatal("findRef should match")
	}
	if got.Kind != RefBranch {
		t.Errorf("findRef picked %s, want branch", got.Kind)
	}
}

func TestSortRefsPrefersProjectGameVersion(t *testing.T) {
	mk := func(name string) RemoteRef {
		r, _ := parseRefName(name)
		r.Kind = RefBranch
		return r
	}
	refs := []RemoteRef{mk("2.0.0/26.3"), mk("1.0.0/26.2"), mk("2.0.0/26.4")}

	// Project is on 26.2 -> the 26.2 ref should win even though 2.0.0/26.4 is
	// a higher version, because "current GameVer" is the documented meaning.
	sortRefs(refs, "26.2", mustSpec(t, "id-system@latest/26.2"))
	// spec pinned a game version, so plain highest-version ordering applies.
	if refs[0].Name != "2.0.0/26.4" {
		t.Errorf("pinned-gameVer sort put %q first, want 2.0.0/26.4", refs[0].Name)
	}

	refs = []RemoteRef{mk("2.0.0/26.3"), mk("1.0.0/26.2"), mk("2.0.0/26.4")}
	sortRefs(refs, "26.2", mustSpec(t, "id-system@latest"))
	if refs[0].Name != "1.0.0/26.2" {
		t.Errorf("soft-gameVer sort put %q first, want 1.0.0/26.2", refs[0].Name)
	}
}

func mustSpec(t *testing.T, in string) LibrarySpec {
	t.Helper()
	s, err := ParseSpec(in)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// mustVer parses a version for test fixtures.
func mustVer(s string) version.Version { return version.Parse(s) }
