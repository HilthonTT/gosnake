package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	// The keybindings for the game
	Keys *Keys `toml:"keys"`
}

func GetConfig(path string) (*Config, error) {
	c := &Config{
		Keys: DefaultKeys(),
	}

	_, err := toml.DecodeFile(path, c)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return c, nil
		}
		return nil, fmt.Errorf("decoding toml file: %w", err)
	}

	return c, nil
}
