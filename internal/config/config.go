package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App AppConfig      `yaml:"app"`
	DB  DatabaseConfig `yaml:"database"`
}

type AppConfig struct {
	Host           string `yaml:"host"`
	Port           string `yaml:"port"`
	IsProduction   bool   `yaml:"is_production"`
	DownMigrations bool   `yaml:"down_migrations"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func New() (*Config, error) {
	b, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	var cfg Config
	if err = yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &cfg, nil
}
