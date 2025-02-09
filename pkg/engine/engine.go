package engine

import (
	"context"
	"sync"

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

func (e *engine) runAndWait(ctx context.Context, wg *sync.WaitGroup) {
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
}

func (e engine) startDevices() {
	e.devices[NetIO_Drop] = newDrop()
	e.devices[NetIO_UDPServer] = newUDPServer(e.conf, NetIO_UDPServer)
	e.devices[NetIO_Local] = tun.New(e.conf, NetIO_Local)
	e.devices[NetIO_Proxy] = newProxy()

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
	egressChan := dev.EgressChan()
	logrus.Infof("Started goroutine reading from %v", name)

	for {
		if ctx.Err() != nil {
			logrus.Infof("Stopped goroutine reading from %v", name)
			return
		}

		pkt.Reset()
		pkt := <-egressChan

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
		outDev.IngressChan() <- pkt
		logrus.Infof("Sent packet %v via %v", pkt, outDevName)
	}
}
