package main

import (
	"os"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/engine"
	"github.com/sirupsen/logrus"
)

const (
	version = "v0.0.1"
)

func main() {
	logrus.Infof("Running uProxy %v", version)
	conf, err := config.FromCmdline()
	if err != nil {
		logrus.WithError(err).Errorf("Failed to parse config file")
		os.Exit(1)
	}

	engineMgr := engine.New(conf)
	err = engineMgr.Start()
	if err != nil {
		logrus.WithError(err).Error("Failure in running server")
		os.Exit(1)
	}
}
