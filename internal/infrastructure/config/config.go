package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	TelegramToken string `json:"telegram_token"`
}

func Load(path string) (_ *Config, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file %q: %w", path, err)
	}

	defer func() {
		closeErr := file.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("close config file %q: %w", path, closeErr)
		}
	}()

	var cfg Config
	decodeErr := json.NewDecoder(file).Decode(&cfg)
	if decodeErr != nil {
		return nil, fmt.Errorf("decode config file %q: %w", path, decodeErr)
	}

	return &cfg, nil
}
