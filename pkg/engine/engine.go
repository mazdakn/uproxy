package engine

import (
	"context"
	"net"
	"sync"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/tunnels/udp"
	"github.com/sirupsen/logrus"
)

type NetIO interface {
	Start(context.Context, *sync.WaitGroup) (int, error)
	WriteChannel() chan<- net.Buffers
}

type engine struct {
	// TODO: change this to a trie struct with IPNet as keys
	peers   map[string]NetIO
	devices []NetIO
	config  *config.Config
}

func New(conf *config.Config) *engine {
	return &engine{
		config: conf,
		peers:  make(map[string]NetIO),
	}
}

func (e *engine) Start(ctx context.Context) error {
	logrus.Info("Starting the engine")

	udpTun := udp.New(e.config)
	e.RegisterDevice(udpTun)

	var wg sync.WaitGroup

	for name, i := range e.devices {
		logrus.Infof("Starting device %v", name)
		n, err := i.Start(ctx, &wg)
		if err != nil {
			logrus.WithError(err).Errorf("Failed to start %v - Skipping", name)
			continue
		}
		logrus.Infof("Successfully started %v with %v goroutines", name, n)
		wg.Add(n)
	}

	wg.Wait()
	return nil
}

func (e *engine) RegisterDevice(dev NetIO) {
	e.devices = append(e.devices, dev)
}

func (e *engine) RegisterPeer(name string, dev NetIO) {
	e.peers[name] = dev
}
