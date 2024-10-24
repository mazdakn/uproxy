package config

import (
	"flag"

	"github.com/sirupsen/logrus"
)

const (
	defaultAddr         string = "0.0.0.0:9999"
	defautMaxBufferSize int    = 1600
)

type Config struct {
	Addr          string
	MaxBufferSize int
}

func newWithDefaults() *Config {
	return &Config{
		MaxBufferSize: defautMaxBufferSize,
	}
}

func FromCmdline() *Config {
	addrPtr := flag.String("addr", defaultAddr, "Address to bind to")

	flag.Parse()
	config := newWithDefaults()
	config.Addr = *addrPtr
	logrus.Infof("Parsed config from command line: %v", config)
	return config
}
