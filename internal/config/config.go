// Package config loads the user's TOML configuration file.
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	// The keybindings for the game.
	Keys *Keys `toml:"keys"`

	// Online leaderboard settings.
	Leaderboard Leaderboard `toml:"leaderboard"`
}

// Leaderboard configures the optional online leaderboard (Tips.Api).
type Leaderboard struct {
	// APIURL is the base URL of the Tips.Api server, e.g.
	// "https://localhost:5001". Empty disables online score submission.
	// Can be overridden with the GOSNAKE_API_URL environment variable.
	APIURL string `toml:"api_url"`

	// InsecureSkipVerify disables TLS certificate verification. Only meant
	// for local development against a self-signed certificate.
	InsecureSkipVerify bool `toml:"insecure_skip_verify"`
}

// Default returns the configuration used when no file exists.
func Default() *Config {
	return &Config{Keys: DefaultKeys()}
}

// GetConfig reads the config at path, falling back to defaults for anything
// the file does not set. A missing file is not an error.
func GetConfig(path string) (*Config, error) {
	c := Default()

	if _, err := toml.DecodeFile(path, c); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("decoding toml file: %w", err)
	}

	if url, ok := os.LookupEnv("GOSNAKE_API_URL"); ok {
		c.Leaderboard.APIURL = url
	}

	return c, nil
}
