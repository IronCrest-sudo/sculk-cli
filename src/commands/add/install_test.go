package add

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sculk-cli/src/commands/initProject/create"
)

func TestGameVersionIncompat(t *testing.T) {
	cases := []struct {
		project, library string
		want             bool
	}{
		// exact
		{"26.2", "26.2", false},
		{"26.3", "26.2", true},
		// the library may declare a range
		{"26.3", ">=26.2", false},
		{"26.1", ">=26.2", true},
		{"1.20.1", "<1.20.1", true},
		{"1.20.0", "<1.20.1", false},
		{"1.21.4", ">=1.20 <26.0", false},
		{"26.2", ">=1.20 <26.0", true},
		// unknown on either side is treated as compatible
		{"", "26.2", false},
		{"26.2", "", false},
		// trailing zeros do not matter
		{"26.2.0", "26.2", false},
	}

	for _, c := range cases {
		got, _, _ := gameVersionIncompat(c.project, c.library)
		if got != c.want {
			t.Errorf("gameVersionIncompat(project=%q, library=%q) = %v, want %v",
				c.project, c.library, got, c.want)
		}
	}
}

func TestAppendTagValuesDedupes(t *testing.T) {
	existing := []string{"a:load", "b:load"}
	incoming := []string{"b:load", "c:load"}

	got := appendTagValues(existing, incoming)
	want := []string{"a:load", "b:load", "c:load"}

	if len(got) != len(want) {
		t.Fatalf("appendTagValues = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("appendTagValues = %v, want %v", got, want)
		}
	}

	// idempotent: merging the same values again changes nothing
	again := appendTagValues(got, incoming)
	if len(again) != len(want) {
		t.Errorf("re-merge grew the tag list: %v", again)
	}
}

func TestSanitizeFolderName(t *testing.T) {
	cases := map[string]string{
		"macroengine": "macroengine",
		"id-system":   "id-system",
		"https://github.com/officialbarden/id-system": "id-system",
		"https://github.com/a/b.git":                  "b",
		"git@github.com:owner/repo.git":               "repo",
		"weird name/with spaces":                      "weird-name-with-spaces",
		"":                                            "sculk-library",
		"///":                                         "sculk-library",
	}
	for in, want := range cases {
		if got := SanitizeFolderName(in); got != want {
			t.Errorf("SanitizeFolderName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSeparateTargetDirRedirectsResourcepacks(t *testing.T) {
	// Pretend we are standing in <world>/datapacks/my_pack
	root := t.TempDir()
	world := filepath.Join(root, "world")
	cwd := filepath.Join(world, "datapacks", "my_pack")
	if err := os.MkdirAll(cwd, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)

	dp, inContainer, err := SeparateTargetDir(VerifyLibraryIntegrity("macroengine"))
	if err != nil {
		t.Fatal(err)
	}
	wantDp := filepath.Join(world, "datapacks", "macroengine")
	if dp != wantDp {
		t.Errorf("datapack target = %s, want %s", dp, wantDp)
	}
	if !inContainer {
		t.Error("a project inside datapacks/ should count as a pack container")
	}

	rp, _, err := SeparateTargetDir(VerifyLibraryIntegrity("macroengine-rp"))
	if err != nil {
		t.Fatal(err)
	}
	wantRp := filepath.Join(world, "resourcepacks", "macroengine-rp")
	if rp != wantRp {
		t.Errorf("resourcepack target = %s, want %s", rp, wantRp)
	}
}

// A project that is not inside a pack container must never cause sculk to
// write into the parent directory, which could be anywhere on disk.
func TestSeparateTargetDirStaysInsideProjectWhenNoContainer(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "somewhere", "my_project")
	if err := os.MkdirAll(cwd, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)

	target, inContainer, err := SeparateTargetDir(VerifyLibraryIntegrity("macroengine"))
	if err != nil {
		t.Fatal(err)
	}
	if inContainer {
		t.Error("a plain project directory should not count as a pack container")
	}
	want := filepath.Join(cwd, "libraries", "macroengine")
	if target != want {
		t.Errorf("target = %s, want %s", target, want)
	}
	if !strings.HasPrefix(target, cwd) {
		t.Errorf("target %s escapes the project directory %s", target, cwd)
	}
}

func TestRegistryLookups(t *testing.T) {
	// every library from the README's todo list must be resolvable
	for _, id := range []string{
		"id-system", "uuid", "astar", "playermotion", "titlewriter",
		"speclib", "reef", "hitmatch", "stringlib", "macroengine", "macroengine-rp",
	} {
		if !IsApproved(id) {
			t.Errorf("registry is missing %q", id)
			continue
		}
		block := VerifyLibraryIntegrity(id)
		if block.Source == "" {
			t.Errorf("%q has no source", id)
		}
		if block.Description == "" {
			t.Errorf("%q has no description", id)
		}
	}

	// lookup must survive a spec suffix and different casing
	if VerifyLibraryIntegrity("MacroEngine").Identifier != "macroengine" {
		t.Error("case-insensitive lookup failed")
	}
	if VerifyLibraryIntegrity("id-system@1.0.0").Source != VerifyLibraryIntegrity("id-system").Source {
		t.Error("lookup with a spec suffix resolved to the wrong source")
	}

	// unknown identifiers stay direct links
	direct := VerifyLibraryIntegrity("https://github.com/someone/thing")
	if direct.Source != "https://github.com/someone/thing" {
		t.Errorf("unknown identifier source = %q, want the identifier itself", direct.Source)
	}

	// the listing must be sorted and de-duplicated
	listed := ListApprovedLibraries()
	seen := map[string]bool{}
	for i, block := range listed {
		if seen[block.Identifier] {
			t.Errorf("duplicate registry entry %q", block.Identifier)
		}
		seen[block.Identifier] = true
		if i > 0 && listed[i-1].Identifier > block.Identifier {
			t.Errorf("registry listing is not sorted at %q", block.Identifier)
		}
	}
}

func TestMacroEngineRegistration(t *testing.T) {
	block := VerifyLibraryIntegrity("macroengine")

	if block.Subdir != "archived/macroEngine-Datapack-v26.4" {
		t.Errorf("macroengine subdir = %q, want archived/macroEngine-Datapack-v26.4", block.Subdir)
	}
	if block.Source != "https://github.com/IronCrest-sudo/core.git" {
		t.Errorf("macroengine source = %q", block.Source)
	}
	if block.packKind() != KindDatapack {
		t.Errorf("macroengine kind = %v, want datapack", block.packKind())
	}
	if block.Notice == "" {
		t.Error("macroengine must warn that it is archived upstream")
	}
	if block.DefaultGameVersion != "26.4-snapshot-1" {
		t.Errorf("macroengine default game version = %q", block.DefaultGameVersion)
	}

	rp := VerifyLibraryIntegrity("macroengine-rp")
	if rp.packKind() != KindResourcepack {
		t.Errorf("macroengine-rp kind = %v, want resourcepack", rp.packKind())
	}
	if rp.Subdir != "packs/macroEngine-Resourcepack-v26.4" {
		t.Errorf("macroengine-rp subdir = %q", rp.Subdir)
	}
}

func TestDefaultPackKindIsDatapack(t *testing.T) {
	block := VerifyLibraryIntegrity("id-system")
	if block.KindOrDefault() != KindDatapack {
		t.Errorf("id-system kind = %v, want datapack", block.KindOrDefault())
	}
}

func TestLibraryRecord(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	r := ResolvedLibrary{
		Block:   libraryBlock{Identifier: "demo", Source: "https://example.test/demo"},
		RepoURL: "https://example.test/demo",
		Meta:    create.LibrariesDotJson{Version: "1.2.3", GameVersion: "26.2"},
		Ref:     RemoteRef{Name: "1.2.3/26.2", Kind: RefBranch, Version: mustVer("1.2.3"), GameVersion: mustVer("26.2")},
		HasRef:  true,
		Mode:    create.InstallSeparate,
	}

	got := libraryRecord(r)
	if got.Identifier != "demo" || got.Version != "1.2.3" || got.GameVersion != "26.2" {
		t.Errorf("unexpected record: %+v", got)
	}
	if got.Install != create.InstallSeparate {
		t.Errorf("record.Install = %q, want separate", got.Install)
	}
	if got.Ref != "1.2.3/26.2" {
		t.Errorf("record.Ref = %q, want 1.2.3/26.2", got.Ref)
	}

	// no ref -> empty label
	r.HasRef = false
	if got := libraryRecord(r); got.Ref != "" {
		t.Errorf("record.Ref = %q, want empty for the default branch", got.Ref)
	}
}
