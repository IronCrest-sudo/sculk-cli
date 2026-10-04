package add

import "testing"

func TestRegistryIdentifierForURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"https://github.com/IronCrest-sudo/core/tree/main/archived/macroEngine-Datapack-v26.4", "macroengine", true},
		{"https://github.com/IronCrest-sudo/core/tree/main/archived/macroEngine-Datapack-v26.4/", "macroengine", true},
		{"github.com/IronCrest-sudo/core/tree/dev/archived/macroEngine-Datapack-v26.4", "macroengine", true},
		{"https://github.com/IronCrest-sudo/core/tree/main/packs/macroEngine-Resourcepack-v26.4", "macroengine-rp", true},
		// A bare monorepo URL does not say which pack is meant.
		{"https://github.com/IronCrest-sudo/core", "", false},
		{"https://github.com/someone/else", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := RegistryIdentifierForURL(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("RegistryIdentifierForURL(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestParseSpecMapsRegisteredTreeURL(t *testing.T) {
	spec, err := ParseSpec("https://github.com/IronCrest-sudo/core/tree/main/archived/macroEngine-Datapack-v26.4")
	if err != nil {
		t.Fatal(err)
	}
	if spec.IsURL || spec.Identifier != "macroengine" {
		t.Errorf("got %+v, want registry identifier 'macroengine' and IsURL=false", spec)
	}

	// Unregistered links stay direct URLs.
	spec, err = ParseSpec("https://github.com/someone/else")
	if err != nil {
		t.Fatal(err)
	}
	if !spec.IsURL {
		t.Errorf("unregistered URL should stay a direct URL, got %+v", spec)
	}
}
