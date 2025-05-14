package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type HarborConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Config struct {
	Harbor HarborConfig `yaml:"harbor"`
}

var Global Config

func LoadConfig(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	if err := yaml.Unmarshal(data, &Global); err != nil {
		log.Fatalf("Failed to unmarshal config file: %v", err)
	}
}
