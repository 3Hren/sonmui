package config

import (
	"io/ioutil"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
)

const (
	DefaultConfigPath = "~/.sonm/icli.yaml"
)

type Config struct {
	AccountPaths map[common.Address]string `yaml:"accounts"`
}

func NewConfig() *Config {
	return &Config{
		AccountPaths: map[common.Address]string{},
	}
}

func LoadConfig(path string) (*Config, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(content, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
