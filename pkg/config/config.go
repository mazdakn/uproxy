package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	defaultFile         string = "config.yaml"
	defaultAddress      string = "0.0.0.0:9999"
	defautMaxBufferSize int    = 1600
	defaultTunName      string = "uproxy"
	defaultMTU          int    = 1400
)

type TunConfig struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
	MTU     int    `yaml:"mtu"`
}

type Policy struct {
	SrcAddr string `yaml:"srcAddr"`
	DstAddr string `yaml:"dstAddr"`
	DstPort string `yaml:"dstPort"`

	Action string `yaml:"action"`
}

type Config struct {
	MaxBufferSize int        `yaml:"maxBufferSize"`
	Address       string     `yaml:"address"`
	Tun           *TunConfig `yaml:"tun"`
	Policies      []Policy   `yaml:"policies"`
}

func ApplyDefaults(config *Config) {
	if config.MaxBufferSize == 0 {
		config.MaxBufferSize = defautMaxBufferSize
	}
	if config.Address == "" {
		config.Address = defaultAddress
	}
	if config.Tun.Name == "" {
		config.Tun.Name = defaultTunName
	}
	if config.Tun.MTU == 0 {
		config.Tun.MTU = defaultMTU
	}
}

func FromCmdline() (*Config, error) {
	filename := flag.String("conf", defaultFile, "Default config file")
	flag.Parse()

	configFile, err := os.ReadFile(*filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %v - err: %w", configFile, err)
	}

	var config Config
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse the config file %v - err: %w", configFile, err)
	}

	logrus.Debugf("Parsed config from command line: %v", config)
	return &config, nil
}
