package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	SMTP  SMTPConfig  `yaml:"smtp"`
	Gmail GmailConfig `yaml:"gmail"`
}

// SMTPConfig represents SMTP server configuration
type SMTPConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// GmailConfig represents Gmail API configuration
type GmailConfig struct {
	CredentialsFile string `yaml:"credentials_file"`
	TokenFile       string `yaml:"token_file"`
}

// Load loads configuration from a YAML file
func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if cfg.SMTP.Host == "" {
		cfg.SMTP.Host = "localhost"
	}
	if cfg.SMTP.Port == 0 {
		cfg.SMTP.Port = 2525
	}
	if cfg.Gmail.TokenFile == "" {
		cfg.Gmail.TokenFile = "token.json"
	}

	return &cfg, nil
}
