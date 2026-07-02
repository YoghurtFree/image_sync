package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig
	Redis      RedisConfig
	Asynq      AsynqConfig
	Registries map[string]RegistryConfig
}

type ServerConfig struct {
	Port int
}

type RedisConfig struct {
	Addr     string
	Password string
}

type AsynqConfig struct {
	Concurrency int
}

type RegistryConfig struct {
	URL      string
	Username string
	Password string
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
