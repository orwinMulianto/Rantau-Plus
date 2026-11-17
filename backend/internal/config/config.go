package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Address      string
	DatabaseURL  string
	AIServiceURL string
	DemoEnabled  bool
}

func Load() (Config, error) {
	cfg := Config{
		Address:      os.Getenv("APP_ADDR"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		AIServiceURL: os.Getenv("AI_SERVICE_URL"),
	}
	if cfg.Address == "" {
		cfg.Address = "127.0.0.1:8080"
	}
	if value, present := os.LookupEnv("RANTAU_DEMO"); present {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("RANTAU_DEMO harus berupa boolean: %w", err)
		}
		cfg.DemoEnabled = enabled
	}
	return cfg, nil
}
