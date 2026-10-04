package config

import (
	"os"
	"testing"
)

// isolate points XDG_CONFIG_HOME at a temp dir so the tests never touch the
// real ~/.config/.sculk/config.json.
func isolate(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
}

func TestInitAndRead(t *testing.T) {
	isolate(t)

	InitConfig()
	cfg, path, err := GetSculkConfig()
	if err != nil {
		t.Fatalf("GetSculkConfig after InitConfig: %v", err)
	}
	if path == "" {
		t.Fatal("config path is empty")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not written: %v", err)
	}
	if !cfg.DoMerge {
		t.Errorf("default doMerge = false, want true")
	}
	if cfg.Author.Name == "" {
		t.Error("default author name is empty")
	}
}

func TestConfigExistsCreates(t *testing.T) {
	isolate(t)

	if ConfigExists() {
		t.Fatal("ConfigExists() = true before any config was written")
	}
	// ConfigExists creates one as a side effect.
	if !ConfigExists() {
		t.Fatal("ConfigExists() = false after it should have created the file")
	}
}

func TestMainSetsDoMerge(t *testing.T) {
	isolate(t)

	for _, c := range []struct {
		arg  string
		want bool
	}{
		{"false", false},
		{"off", false},
		{"0", false},
		{"no", false},
		{"true", true},
		{"on", true},
		{"1", true},
		{"yes", true},
	} {
		if err := Main([]string{"doMerge", c.arg}); err != nil {
			t.Fatalf("Main(doMerge %s): %v", c.arg, err)
		}
		if got := MustConfig().DoMerge; got != c.want {
			t.Errorf("doMerge %s -> %v, want %v", c.arg, got, c.want)
		}
		if got := ShouldMerge(); got != c.want {
			t.Errorf("ShouldMerge() after doMerge %s = %v, want %v", c.arg, got, c.want)
		}
	}
}

func TestMainRejectsBadBool(t *testing.T) {
	isolate(t)
	InitConfig()

	if err := Main([]string{"doMerge", "maybe"}); err == nil {
		t.Error("Main(doMerge maybe) should fail")
	}
}

func TestMainSetsAuthor(t *testing.T) {
	isolate(t)

	if err := Main([]string{"author", "IronCrest"}); err != nil {
		t.Fatalf("Main(author): %v", err)
	}
	if got := MustConfig().Author.Name; got != "IronCrest" {
		t.Errorf("author = %q, want IronCrest", got)
	}
	if got := AuthorName(); got != "IronCrest" {
		t.Errorf("AuthorName() = %q, want IronCrest", got)
	}

	// multi-word author names are joined back together
	if err := Main([]string{"author", "The", "Official", "Barden"}); err != nil {
		t.Fatalf("Main(author multiword): %v", err)
	}
	if got := MustConfig().Author.Name; got != "The Official Barden" {
		t.Errorf("author = %q, want 'The Official Barden'", got)
	}

	if err := Main([]string{"author", "   "}); err == nil {
		t.Error("Main(author '') should fail")
	}
}

func TestMainSetsInitTemplateAndSculkVersion(t *testing.T) {
	isolate(t)

	if err := Main([]string{"initTemplate", "flat"}); err != nil {
		t.Fatalf("Main(initTemplate): %v", err)
	}
	if got := MustConfig().InitTemplate; got != "flat" {
		t.Errorf("initTemplate = %q, want flat", got)
	}

	if err := Main([]string{"sculk.version", "1.2.0"}); err != nil {
		t.Fatalf("Main(sculk.version): %v", err)
	}
	if got := MustConfig().Sculk.Version; got != "1.2.0" {
		t.Errorf("sculk.version = %q, want 1.2.0", got)
	}
}

func TestMainUnknownParam(t *testing.T) {
	isolate(t)

	if err := Main([]string{"nope", "x"}); err == nil {
		t.Error("Main(nope x) should fail")
	}
	if err := Main([]string{"nope"}); err == nil {
		t.Error("Main(nope) should fail")
	}
}

func TestMainResetAndList(t *testing.T) {
	isolate(t)
	InitConfig()

	if err := Main([]string{"doMerge", "false"}); err != nil {
		t.Fatal(err)
	}
	if MustConfig().DoMerge {
		t.Fatal("doMerge should be false before reset")
	}

	if err := Main([]string{"reset"}); err != nil {
		t.Fatalf("Main(reset): %v", err)
	}
	if !MustConfig().DoMerge {
		t.Error("reset should restore doMerge = true")
	}

	// these must not error and must not change anything
	for _, args := range [][]string{{}, {"list"}, {"help"}} {
		if err := Main(args); err != nil {
			t.Errorf("Main(%v): %v", args, err)
		}
	}
}

func TestParseBool(t *testing.T) {
	for _, s := range []string{"true", "TRUE", "yes", "on", "1", "enable"} {
		if v, err := parseBool(s); err != nil || !v {
			t.Errorf("parseBool(%q) = %v, %v; want true, nil", s, v, err)
		}
	}
	for _, s := range []string{"false", "FALSE", "no", "off", "0", "disable"} {
		if v, err := parseBool(s); err != nil || v {
			t.Errorf("parseBool(%q) = %v, %v; want false, nil", s, v, err)
		}
	}
	if _, err := parseBool("wat"); err == nil {
		t.Error("parseBool(wat) should fail")
	}
}
