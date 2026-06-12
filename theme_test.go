package main

import "testing"

func TestParseSystemThemeResolvesAliasesAndDefaults(t *testing.T) {
	colors, err := parseSystemTheme([]byte(`
# Test theme
type=dark
bg=000000
fg=f4f4f9
dark=2f4550
medium=577787
light=b8dbd9
accent=$light
success=98c379
`))
	if err != nil {
		t.Fatalf("parseSystemTheme returned error: %v", err)
	}

	assertColor := func(key, want string) {
		t.Helper()
		if got := colors[key]; got != want {
			t.Fatalf("colors[%q] = %q, want %q", key, got, want)
		}
	}

	assertColor("bg", "#000000")
	assertColor("fg", "#F4F4F9")
	assertColor("accent", "#B8DBD9")
	assertColor("surface", "#2F4550")
	assertColor("text", "#F4F4F9")
	assertColor("text_dim", "#577787")
	assertColor("success", "#98C379")
}

func TestParseSystemThemeAcceptsHashColorsAndInlineComments(t *testing.T) {
	colors, err := parseSystemTheme([]byte(`
bg=#fff # inline comment
fg=#111
dark=#eee
medium=#777
light=#06c
`))
	if err != nil {
		t.Fatalf("parseSystemTheme returned error: %v", err)
	}

	if colors["bg"] != "#FFFFFF" {
		t.Fatalf("colors[bg] = %q, want #FFFFFF", colors["bg"])
	}
	if colors["light"] != "#0066CC" {
		t.Fatalf("colors[light] = %q, want #0066CC", colors["light"])
	}
}

func TestParseSystemThemeRequiresCorePalette(t *testing.T) {
	_, err := parseSystemTheme([]byte(`
bg=000000
fg=ffffff
`))
	if err == nil {
		t.Fatal("parseSystemTheme returned nil error for incomplete theme")
	}
}
