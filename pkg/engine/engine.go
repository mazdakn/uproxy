package engine

import (
	"context"
	"net"
	"sync"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/tun"
	"github.com/mazdakn/uproxy/pkg/udp"
	"github.com/sirupsen/logrus"
)

type NetIO interface {
	Start(context.Context, *sync.WaitGroup) (int, error)
	WriteChannel() chan<- net.Buffers
	Name() string
}

type engine struct {
	// TODO: change this to a trie struct with IPNet as keys
	peers   map[string]NetIO
	devices []NetIO
	conf    *config.Config
}

func New(conf *config.Config) *engine {
	return &engine{
		conf:  conf,
		peers: make(map[string]NetIO),
	}
}

func (e *engine) Start(ctx context.Context) error {
	logrus.Info("Starting the engine")

	udpTunnel := udp.New(e.conf)
	e.RegisterDevice(udpTunnel)

	tunDev := tun.New(e.conf)
	e.RegisterDevice(tunDev)

	var wg sync.WaitGroup

	for _, dev := range e.devices {
		name := dev.Name()
		logrus.Infof("Starting device %v", name)
		n, err := dev.Start(ctx, &wg)
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
