package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sources []SourceConfig `yaml:"sources"`
}

type SourceConfig struct {
	Name            string   `yaml:"name"`
	Type            string   `yaml:"type"`
	Mode            string   `yaml:"mode"`
	URL             string   `yaml:"url"`
	Enabled         bool     `yaml:"enabled"`
	IncludeKeywords []string `yaml:"include_keywords"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal yaml: %w", err)
	}
	return cfg, nil
}
