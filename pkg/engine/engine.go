package engine

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/packet"
	"github.com/mazdakn/uproxy/pkg/tun"
	"github.com/sirupsen/logrus"
)

type engine struct {
	conf *config.Config

	tunDev    *tun.TunDevice
	dropDev   *dropDevice
	udpServer *udpServer
	proxy     NetIO

	policies []Policy
}

func New(conf *config.Config) *engine {
	return &engine{
		conf: conf,
	}
}

func (e *engine) Run() error {
	ctx, cancelFunc := setupSignals()
	defer cancelFunc()

	defer e.cleanup()

	var wg sync.WaitGroup
	logrus.Info("Starting the engine")

	e.udpServer = newUDPServer(e.conf)
	err := e.initDevice(ctx, e.udpServer, &wg)
	if err != nil {
		return err
	}

	e.dropDev = newDrop()

	//e.proxy = proxy.New()

	if e.conf.Tun != nil {
		e.tunDev = tun.New(e.conf)
		err := e.initDevice(ctx, e.tunDev, &wg)
		if err != nil {
			return err
		}
	}

	err = e.ParsePolicies()
	if err != nil {
		return err
	}

	wg.Wait()
	return nil
}

func (e *engine) initDevice(ctx context.Context, dev NetIO, wg *sync.WaitGroup) error {
	name := dev.Name()
	logrus.Infof("Starting device %v", name)
	if err := dev.Start(); err != nil {
		return err
	}
	wg.Add(2)
	go e.devWriter(ctx, dev, wg)
	go e.devReader(ctx, dev, wg)
	logrus.Infof("Successfully started %v", name)
	return nil
}

func (e *engine) cleanup() {
	if e.tunDev != nil {
		// Clean up tun device
		err := e.tunDev.Stop()
		if err != nil {
			logrus.WithError(err).Errorf("Failed cleaning up %v", e.tunDev.Name())
		}
	}
}

func (e *engine) ParsePolicies() error {
	for _, p := range e.conf.Policies {
		if p.SrcAddr == "" && p.DstAddr == "" && p.DstPort == "" {
			logrus.Errorf("No match provided: %v - Skipping.", p)
			continue
		}
		if p.Action == "" {
			logrus.Errorf("No action provided: %v - Skipping.", p)
			continue
		}

		var err error
		var rPolicy Policy
		rPolicy.Action, rPolicy.Endpoint, err = e.policyAction(p.Action)
		if err != nil {
			logrus.WithError(err).Errorf("Error parsing action: %v - Skipping", p.Action)
			continue
		}
		if p.SrcAddr != "" {
			_, rPolicy.SrcNet, err = net.ParseCIDR(p.SrcAddr)
			if err != nil {
				logrus.WithError(err).Errorf("Invalid source cidr %v - Skipping", p.SrcAddr)
				continue
			}
		}
		if p.DstAddr != "" {
			_, rPolicy.DstNet, err = net.ParseCIDR(p.DstAddr)
			if err != nil {
				logrus.WithError(err).Errorf("Invalid destination cidr %v - Skipping", p.DstAddr)
				continue
			}
		}
		rPolicy.Proto, rPolicy.DstPort = policyProtoPort(p.DstPort)

		logrus.Debugf("Adding policy %#v", rPolicy)
		e.policies = append(e.policies, rPolicy)
	}

	return nil
}

func (e engine) policyAction(action string) (NetIO, *net.UDPAddr, error) {
	// Need to handle route action separately
	if strings.HasPrefix(action, string(ActionRoute)) {
		addr := strings.TrimLeft(action, "route=")
		udpAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return nil, nil, err
		}
		return e.udpServer, udpAddr, nil
	}

	switch Action(action) {
	case ActionLocal:
		if e.tunDev == nil {
			return nil, nil, fmt.Errorf("local device not available")
		}
		return e.tunDev, nil, nil
	case ActionProxy:
		return e.proxy, nil, nil
	case ActionDrop:
		return e.dropDev, nil, nil
	}
	return nil, nil, fmt.Errorf("failed to parse action %v", action)
}

func (e engine) MatchPolicies(pkt *packet.Packet) *Policy {
	logrus.Debugf("Looking up packet %v", pkt)
	for _, p := range e.policies {
		if e.MatchPolicy(p, pkt) {
			return &p
		}
	}
	return nil
}

func (e engine) MatchPolicy(policy Policy, pkt *packet.Packet) bool {
	if policy.DstNet != nil && !policy.DstNet.Contains(pkt.DstAddr()) {
		return false
	}
	if policy.SrcNet != nil && !policy.SrcNet.Contains(pkt.SrcAddr()) {
		return false
	}
	if policy.Proto != 0 && policy.Proto != pkt.Protocol() {
		return false
	}
	if policy.DstPort != 0 && policy.DstPort != pkt.DstPort() {
		return false
	}
	return true
}

func (e *engine) devReader(ctx context.Context, dev NetIO, wg *sync.WaitGroup) {
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
			logrus.Infof("Received %v bytes from %v.", num, name)
			err = pkt.Parse(num)
			if err != nil {
				logrus.WithError(err).Error("Failed to parse packet")
				continue
			}
			logrus.Infof("Packet : %v", pkt)

			policy := e.MatchPolicies(pkt)
			if policy == nil {
				logrus.Warnf("not policy found")
				continue
			}
			logrus.Debugf("Sending packet to %v via endpoint %v", policy.Endpoint, policy.Action.Name())
			if policy.Endpoint != nil {
				pkt.Endpoint = policy.Endpoint
			}
			writeC := policy.Action.Channel()
			*writeC <- pkt
		}
	}
}

func (e *engine) devWriter(ctx context.Context, dev NetIO, wg *sync.WaitGroup) {
	defer wg.Done()
	name := dev.Name()
	logrus.Infof("Started goroutine writing to %v", name)
	var err error
	var num int
	devChan := dev.Channel()
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
