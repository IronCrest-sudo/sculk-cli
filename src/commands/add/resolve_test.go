package add

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newGitRepo returns a working git checkout with a fixed identity.
func newGitRepo(t *testing.T) (dir string, run func(args ...string)) {
	t.Helper()

	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not available")
	}

	dir = t.TempDir()
	run = func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitBin, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.test",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.test",
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "-q", "-b", "main")
	run("config", "user.email", "t@example.test")
	run("config", "user.name", "t")
	return dir, run
}

func commitRef(t *testing.T, dir string, run func(...string), ref string) {
	t.Helper()
	run("checkout", "-q", "-B", ref)
	body := `{"author":"t","version":"` + ref + `","game_version":"","libraries":[]}`
	if err := os.WriteFile(filepath.Join(dir, "libraries.json"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "--allow-empty", "-m", ref)
}

// buildSlashRefRepo publishes the "1.0.0/26.2" spelling. Because git stores
// refs as files, this repository must NOT also publish a bare "1.0.0" branch.
func buildSlashRefRepo(t *testing.T) string {
	dir, run := newGitRepo(t)
	commitRef(t, dir, run, "main")
	commitRef(t, dir, run, "1.0.0/26.2")
	commitRef(t, dir, run, "2.0.0/26.4")
	run("tag", "v1.5.0", "1.0.0/26.2")
	ClearRefCache()
	return dir
}

// buildFlatRefRepo publishes the git-safe flat spellings, which CAN coexist
// with a bare "1.0.0" branch.
func buildFlatRefRepo(t *testing.T) string {
	dir, run := newGitRepo(t)
	commitRef(t, dir, run, "main")
	commitRef(t, dir, run, "1.0.0")
	commitRef(t, dir, run, "1.0.0_26.2")
	commitRef(t, dir, run, "2.0.0+26.4")
	run("tag", "v1.5.0", "1.0.0")
	ClearRefCache()
	return dir
}

func TestGitRefCollisionIsReal(t *testing.T) {
	// Documents the constraint the separator list exists to work around: git
	// refuses refs/heads/1.0.0/26.2 once refs/heads/1.0.0 exists.
	dir, run := newGitRepo(t)
	commitRef(t, dir, run, "main")
	commitRef(t, dir, run, "1.0.0")

	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not available")
	}
	cmd := exec.Command(gitBin, "checkout", "-q", "-B", "1.0.0/26.2")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Errorf("git unexpectedly allowed the colliding ref:\n%s", out)
	}
}

func TestResolveRefAgainstRealGit(t *testing.T) {
	cases := []struct {
		name      string
		slashRepo bool
		arg       string
		projectGV string
		wantRef   string // "" means "default branch, nothing pinned"
	}{
		{"plain identifier uses the default branch", true, "demo", "26.2", ""},
		{"@latest uses the default branch", true, "demo@latest", "26.2", ""},

		// README examples, slash spelling
		{"@1.0.0/26.2 pins both", true, "demo@1.0.0/26.2", "26.3", "1.0.0/26.2"},
		{"@/26.4 picks the newest version for that GameVer", true, "demo@/26.4", "26.4", "2.0.0/26.4"},
		{"@1.0.0 finds 1.0.0/26.2 via the soft GameVer preference", true, "demo@1.0.0", "26.2", "1.0.0/26.2"},

		// README examples, flat spelling
		{"@1.0.0 resolves the underscore ref", false, "demo@1.0.0", "26.2", "1.0.0_26.2"},
		{"@1.0.0/26.2 accepts the underscore ref", false, "demo@1.0.0/26.2", "26.3", "1.0.0_26.2"},
		{"@/26.4 finds the plus ref", false, "demo@/26.4", "26.4", "2.0.0+26.4"},
		{"bare version ref is reachable", false, "demo@1.0.0", "26.9", "1.0.0"},

		// constraint forms. The slash repo also publishes a v1.5.0 tag, which
		// is the highest 1.x ref, so a range that admits it must pick it.
		{"@^1.0.0 stays inside 1.x", true, "demo@^1.0.0", "26.9", "v1.5.0"},
		{"@>=2.0.0 reaches 2.0.0/26.4", true, "demo@>=2.0.0", "26.9", "2.0.0/26.4"},
		{"@<2.0.0 excludes 2.0.0/26.4", true, "demo@<2.0.0", "26.9", "v1.5.0"},
		// the flat repo has the same tag but no 1.x ref above 1.0.0
		{"@^1.0.0 in the flat repo", false, "demo@^1.0.0", "26.9", "v1.5.0"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := buildSlashRefRepo(t)
			if !c.slashRepo {
				repo = buildFlatRefRepo(t)
			}
			ClearRefCache()

			spec, err := ParseSpec(c.arg)
			if err != nil {
				t.Fatal(err)
			}

			ref, hasRef, err := ResolveRef(repo, spec, c.projectGV)
			if err != nil {
				t.Fatalf("ResolveRef(%q): %v", c.arg, err)
			}

			if c.wantRef == "" {
				if hasRef {
					t.Fatalf("ResolveRef(%q) pinned %q, want the default branch", c.arg, ref.Name)
				}
				return
			}
			if !hasRef {
				t.Fatalf("ResolveRef(%q) did not pin a ref, want %q", c.arg, c.wantRef)
			}
			if ref.Name != c.wantRef {
				t.Errorf("ResolveRef(%q) = %q, want %q", c.arg, ref.Name, c.wantRef)
			}
		})
	}
}

func TestResolveRefReportsAvailableRefsWhenNothingMatches(t *testing.T) {
	repo := buildSlashRefRepo(t)
	ClearRefCache()

	spec, err := ParseSpec("demo@9.9.9")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = ResolveRef(repo, spec, "26.2")
	if err == nil {
		t.Fatal("expected an error for an unsatisfiable version")
	}

	msg := err.Error()
	for _, want := range []string{"9.9.9", "1.0.0", "2.0.0"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error should mention %q, got: %s", want, msg)
		}
	}
}

func TestListRemoteRefsIgnoresNonVersionBranches(t *testing.T) {
	repo := buildFlatRefRepo(t)
	ClearRefCache()

	refs, err := ListRemoteRefs(repo)
	if err != nil {
		t.Fatal(err)
	}

	names := map[string]bool{}
	for _, r := range refs {
		names[r.Name] = true
	}

	if names["main"] {
		t.Errorf("'main' was treated as a version: %v", names)
	}
	for _, want := range []string{"1.0.0", "1.0.0_26.2", "2.0.0+26.4", "v1.5.0"} {
		if !names[want] {
			t.Errorf("expected ref %q in %v", want, names)
		}
	}
}

func TestSplitRefName(t *testing.T) {
	cases := []struct {
		in, ver, gv string
	}{
		{"1.0.0", "1.0.0", ""},
		{"1.0.0/26.2", "1.0.0", "26.2"},
		{"1.0.0+26.2", "1.0.0", "26.2"},
		{"1.0.0_26.2", "1.0.0", "26.2"},
		// a hyphen is part of the version, never a separator
		{"26.4-snapshot-1", "26.4-snapshot-1", ""},
		{"1.0.0-rc1", "1.0.0-rc1", ""},
		// ...but it is fine inside the game-version half
		{"1.0.0/26.4-snapshot-1", "1.0.0", "26.4-snapshot-1"},
		{"1.0.0_26.4-snapshot-1", "1.0.0", "26.4-snapshot-1"},
	}
	for _, c := range cases {
		ver, gv := splitRefName(c.in)
		if ver != c.ver || gv != c.gv {
			t.Errorf("splitRefName(%q) = (%q, %q), want (%q, %q)", c.in, ver, gv, c.ver, c.gv)
		}
	}
}
