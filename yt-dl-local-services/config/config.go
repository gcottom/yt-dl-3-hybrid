package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

func LoadConfigFromFile(path string) (*Config, error) {
	if path == "" {
		path = "./config/config.yaml"
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var config Config
	dec := yaml.NewDecoder(file)
	err = dec.Decode(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

type Config struct {
	LambdaDomain        string `yaml:"lambda_domain"`
	LocalPort           int    `yaml:"local_port"`
	ConcurrentDownloads int    `yaml:"concurrent_downloads"`
	SaveDir             string `yaml:"save_dir"`
}
