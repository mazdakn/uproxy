package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/engine"
	"github.com/sirupsen/logrus"
)

const (
	version = "v0.0.1"
)

func setupSignals(cancelFunc context.CancelFunc) {
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		signal := <-signalC
		logrus.Infof("Received singal %v", signal)
		cancelFunc()
	}()
}

func main() {
	logrus.Infof("Running uProxy %v", version)
	conf, err := config.FromCmdline()
	if err != nil {
		logrus.WithError(err).Errorf("Failed to parse config file")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignals(cancel)

	engineMgr := engine.New(conf)
	err = engineMgr.Start(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failure in running server")
		os.Exit(1)
	}
}
