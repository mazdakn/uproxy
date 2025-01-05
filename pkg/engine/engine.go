package engine

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/packet"
	"github.com/mazdakn/uproxy/pkg/tun"
	"github.com/sirupsen/logrus"
)

type engine struct {
	conf     *config.Config
	devices  []NetIO
	policies *PolicyTable
}

func New(conf *config.Config) *engine {
	return &engine{
		conf:     conf,
		policies: newPolicyTable(),
		devices:  make([]NetIO, NetIO_Max),
	}
}

func (e *engine) Run() error {
	logrus.Info("Starting the engine")
	ctx, cancelFunc := setupSignals()
	defer cancelFunc()

	defer e.cleanup()

	err := e.policies.ParseConfig(e.conf)
	if err != nil {
		return err
	}

	e.startDevices()
	logrus.Info("Started the engine")

	var wg sync.WaitGroup
	e.runAndWait(ctx, &wg)
	return nil
}

func (e *engine) runAndWait(ctx context.Context, wg *sync.WaitGroup) error {
	udpDev := e.devices[NetIO_UDPServer]
	if udpDev != nil {
		wg.Add(1)
		go e.handleDevice(ctx, udpDev, wg)
	}

	tunDev := e.devices[NetIO_Local]
	if tunDev != nil {
		wg.Add(1)
		go e.handleDevice(ctx, tunDev, wg)
	}

	wg.Wait()
	return nil
}

func (e engine) startDevices() {
	e.devices[NetIO_UDPServer] = newUDPServer(e.conf)
	e.devices[NetIO_Drop] = newDrop()
	e.devices[NetIO_Proxy] = newProxy()
	e.devices[NetIO_Local] = tun.New(e.conf)

	for i, dev := range e.devices {
		if dev == nil {
			logrus.Debugf("no device found at index %v", i)
			continue
		}
		err := dev.Start()
		if err != nil {
			logrus.WithError(err).Warnf("failed to start device %v", dev.Name())
			continue
		}
		logrus.Infof("Successfully started %v", dev.Name())
	}
}

func (e *engine) cleanup() {
	for _, dev := range e.devices {
		if dev == nil {
			continue
		}
		err := dev.Stop()
		if err != nil {
			logrus.WithError(err).Errorf("Failed cleaning up %v", dev.Name())
		}
	}
}

func (e *engine) handleDevice(ctx context.Context, dev NetIO, wg *sync.WaitGroup) {
	defer wg.Done()
	name := dev.Name()
	pkt := packet.New(e.conf.MaxBufferSize)
	logrus.Infof("Started goroutine reading from %v", name)
	for {
		pkt.Reset()
		select {
		case <-ctx.Done():
			logrus.Infof("Stopped goroutine reading from %v", name)
			return
		default:
			num, err := dev.Read(pkt, time.Now().Add(time.Second))
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

			err = pkt.Parse(num)
			if err != nil {
				logrus.WithError(err).Error("Failed to parse packet")
				continue
			}
			logrus.Infof("Packet : %v", pkt)

			policy := e.policies.Match(pkt)
			if policy == nil {
				logrus.Warnf("not policy found")
				continue
			}

			outDevIdx := policy.Action
			outDev := e.devices[outDevIdx]
			if outDev == nil {
				logrus.Warnf("target device at index %v not available", outDevIdx)
				continue
			}
			outDevName := outDev.Name()
			logrus.Infof("Sending packet to %v via endpoint %v", policy.Endpoint, outDevName)

			if policy.Endpoint != nil {
				pkt.Meta.Endpoint = policy.Endpoint
			}

			// Write Packet
			num, err = outDev.Write(pkt, time.Now().Add(time.Second))
			if err != nil {
				logrus.WithError(err).Errorf("Failed to write to %v", outDevName)
				continue
			}
			if num != pkt.Len() {
				logrus.Errorf("Error in writing packet to %v", outDevName)
				continue
			}
			logrus.Infof("Sent packet %v via %v", pkt, outDevName)
		}
	}
}
