package engine

import (
	"context"
	"net"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/packet"
	"github.com/mazdakn/uproxy/pkg/routing"
	"github.com/mazdakn/uproxy/pkg/tun"
	"github.com/mazdakn/uproxy/pkg/udp"
	"github.com/sirupsen/logrus"
)

type engine struct {
	routeTable *routing.RouteTabel
	conf       *config.Config
}

func New(conf *config.Config) *engine {
	return &engine{
		conf:       conf,
		routeTable: routing.New(conf),
	}
}

func (e *engine) Start() error {
	ctx, cancelFunc := setupSignals()
	defer cancelFunc()

	var wg sync.WaitGroup
	logrus.Info("Starting the engine")

	udpTunnel := udp.New(e.conf)
	e.initDevice(ctx, udpTunnel, &wg)

	// TODO: make creating tun device optional based on configs
	tunDev := tun.New(e.conf)
	e.initDevice(ctx, tunDev, &wg)

	err := e.routeTable.ParseRoutes()
	if err != nil {
		return err
	}

	wg.Wait()
	return nil
}

func (e *engine) initDevice(ctx context.Context, dev routing.NetIO, wg *sync.WaitGroup) error {
	name := dev.Name()
	logrus.Infof("Starting device %v", name)
	if err := dev.Start(ctx, wg); err != nil {
		return err
	}
	e.routeTable.RegisterDevice(dev)
	wg.Add(2)
	go e.devWriter(ctx, dev, wg)
	go e.devReader(ctx, dev, wg)
	logrus.Infof("Successfully started %v", name)
	return nil
}

func (e *engine) devReader(ctx context.Context, dev routing.NetIO, wg *sync.WaitGroup) {
	defer wg.Done()
	name := dev.Name()
	logrus.Infof("Started goroutine reading from %v", name)
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("Stopped goroutine reading from %v", name)
			return
		default:
			pkt := packet.New(e.conf.MaxBufferSize)
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
			logrus.Debugf("Received %v bytes from %v.", num, name)
			err = pkt.Parse(num)
			if err != nil {
				logrus.WithError(err).Error("Failed to parse packet")
				continue
			}
			logrus.Debugf("Packet : %v", pkt)

			route := e.routeTable.Lookup(pkt.DstAddr())
			if route == nil {
				logrus.Warnf("not route entry found")
				continue
			}
			logrus.Debugf("Sending packet to %v via endpoint %v", route.Endpoint, route.Device.Name())
			if route.Endpoint != nil {
				pkt.Endpoint = route.Endpoint
			}
			writeC := route.Device.WriteC()
			*writeC <- pkt
		}
	}
}

func (e *engine) devWriter(ctx context.Context, dev routing.NetIO, wg *sync.WaitGroup) {
	defer wg.Done()
	name := dev.Name()
	logrus.Infof("Started goroutine writing to %v", name)
	var err error
	var num int
	devChan := dev.WriteC()
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("Stoped goroutine writing to %v", name)
			return
		case packets := <-*devChan:
			num, err = dev.Write(packets, time.Now().Add(time.Second))
			if err != nil {
				logrus.Errorf("Failed to write to %v", name)
				continue
			}
			if num != packets.Len() {
				logrus.Errorf("Error in writing packet to %v", dev.Name())
				continue
			}
			logrus.Debugf("Sent packet %v via %v", packets, name)
		}
	}
}

func setupSignals() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}
