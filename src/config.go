package src

import (
	"os"

	"gopkg.in/yaml.v2"
)

type ConnectionConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type MetricsConfig struct {
	Port int `yaml:"port"`
}

type Config struct {
	Postgres    ConnectionConfig `yaml:"postgres"`
	SingleStore ConnectionConfig `yaml:"singlestore"`
	Metrics     MetricsConfig    `yaml:"metrics"`
}

func ParseConfig(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	var cfg Config
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
