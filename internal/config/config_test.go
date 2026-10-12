package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestGetConfigMissingFileUsesDefaults(t *testing.T) {
	t.Setenv("GOSNAKE_API_URL", "")
	os.Unsetenv("GOSNAKE_API_URL")

	c, err := GetConfig(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(c.Keys.Up, DefaultKeys().Up) {
		t.Fatalf("Up = %v, want defaults", c.Keys.Up)
	}
	if c.Leaderboard.APIURL != "" {
		t.Fatal("online leaderboard should be disabled by default")
	}
}

func TestGetConfigOverridesOnlyWhatIsSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := "[keys]\nup = [\"i\"]\n\n[leaderboard]\napi_url = \"https://example.test\"\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	c, err := GetConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(c.Keys.Up, []string{"i"}) {
		t.Fatalf("Up = %v, want [i]", c.Keys.Up)
	}
	if !slices.Equal(c.Keys.Down, DefaultKeys().Down) {
		t.Fatalf("Down = %v, want defaults", c.Keys.Down)
	}
	if c.Leaderboard.APIURL != "https://example.test" {
		t.Fatalf("APIURL = %q", c.Leaderboard.APIURL)
	}

	t.Setenv("GOSNAKE_API_URL", "http://override.test")
	c, err = GetConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Leaderboard.APIURL != "http://override.test" {
		t.Fatalf("env override ignored: %q", c.Leaderboard.APIURL)
	}
}

func TestGetConfigInvalidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("keys = ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := GetConfig(path); err == nil {
		t.Fatal("expected a decode error")
	}
}
