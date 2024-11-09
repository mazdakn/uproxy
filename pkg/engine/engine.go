package engine

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/packet"
	"github.com/mazdakn/uproxy/pkg/tun"
	"github.com/mazdakn/uproxy/pkg/udp"
	"github.com/sirupsen/logrus"
)

type NetIO interface {
	Start() error
	Name() string
	Backend() io.ReadWriter
	SetReadDeadline(time.Time) error
	WriteC() chan net.Buffers
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
		err := dev.Start()
		if err != nil {
			logrus.WithError(err).Errorf("Failed to start %v - Skipping", name)
			continue
		}

		wg.Add(2)
		go e.DeviceReader(ctx, dev, &wg)
		go e.DeviceWrite(ctx, dev, &wg)
		logrus.Infof("Successfully started %v", name)
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

func (e *engine) DeviceReader(ctx context.Context, dev NetIO, wg *sync.WaitGroup) {
	defer wg.Done()
	name := dev.Name()
	logrus.Infof("Started goroutine reading from %v", name)
	buffer := make([]byte, e.conf.MaxBufferSize) // conf.MaxBufferSize
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("Stopped goroutine reading from %v", name)
			return
		default:
			err := dev.SetReadDeadline(time.Now().Add(time.Second))
			if err != nil {
				logrus.Errorf("Failed to set read deadline")
			}
			num, err := dev.Backend().Read(buffer)
			if err != nil {
				nerr, ok := err.(net.Error)
				if ok && !nerr.Timeout() {
					logrus.Errorf("failure in reading from %v", name)
				}
			}
			// Nothing recived.
			if num == 0 {
				continue
			}
			logrus.Infof("Received %v bytes from %v.", num, name)
			packet.Parse(buffer[:num])
		}
	}
}

func (e *engine) DeviceWrite(ctx context.Context, dev NetIO, wg *sync.WaitGroup) {
	defer wg.Done()
	name := dev.Name()
	logrus.Infof("Started goroutine writing to %v", name)
	var err error
	var num int64
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("Stoped goroutine writing to %v", name)
			return
		case packets := <-dev.WriteC():
			num, err = packets.WriteTo(dev.Backend())
			if err != nil {
				logrus.Errorf("Failed to write to %v", name)
				continue
			}
			logrus.Debugf("Sent %v packets via %v", num, name)
		}
	}
}
