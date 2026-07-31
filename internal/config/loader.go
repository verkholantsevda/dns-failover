package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {

	cfg := &Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil

}