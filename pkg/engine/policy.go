package engine

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/packet"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

type Action string

const (
	ActionDrop  Action = "drop"
	ActionProxy Action = "proxy"
	ActionRoute Action = "route"
	ActionLocal Action = "local"
)

type Policy struct {
	SrcNet *net.IPNet
	DstNet *net.IPNet

	Proto   byte
	DstPort uint16

	Action   uint8
	Endpoint *net.UDPAddr
}

func (p Policy) Match(pkt *packet.Packet) bool {
	if p.DstNet != nil && !p.DstNet.Contains(pkt.DstAddr()) {
		return false
	}
	if p.SrcNet != nil && !p.SrcNet.Contains(pkt.SrcAddr()) {
		return false
	}
	if p.Proto != 0 && p.Proto != pkt.Protocol() {
		return false
	}
	if p.DstPort != 0 && p.DstPort != pkt.DstPort() {
		return false
	}
	return true
}

type PolicyTable struct {
	policies []Policy
}

func newPolicyTable() *PolicyTable {
	return &PolicyTable{}
}

func (t *PolicyTable) ParseConfig(conf *config.Config) error {
	for _, p := range conf.Policies {
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
		rPolicy.Action, rPolicy.Endpoint, err = policyAction(p.Action)
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
		t.policies = append(t.policies, rPolicy)
	}

	return nil
}

func (p PolicyTable) Match(pkt *packet.Packet) *Policy {
	logrus.Debugf("Looking up packet %v", pkt)
	for _, p := range p.policies {
		if p.Match(pkt) {
			return &p
		}
	}
	return nil
}

func policyAction(action string) (uint8, *net.UDPAddr, error) {
	// Need to handle route action separately
	if strings.HasPrefix(action, string(ActionRoute)) {
		addr := strings.TrimLeft(action, "route=")
		udpAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return 0, nil, err
		}
		return NetIO_UDPServer, udpAddr, nil
	}

	switch Action(action) {
	case ActionDrop:
		return NetIO_Drop, nil, nil
	case ActionLocal:
		// TODO: need to check this somewhere else
		/*if e.tunDev == nil {
			return nil, nil, fmt.Errorf("local device not available")
		}*/
		return NetIO_Local, nil, nil
	case ActionProxy:
		return NetIO_Proxy, nil, nil
	}
	return 0, nil, fmt.Errorf("failed to parse action %v", action)
}

func policyProtoPort(port string) (byte, uint16) {
	if strings.HasPrefix(port, "udp:") {
		return unix.IPPROTO_UDP, strToPort(strings.TrimLeft(port, "udp:"))
	}
	if strings.HasPrefix(port, "tcp:") {
		return unix.IPPROTO_TCP, strToPort(strings.TrimLeft(port, "tcp:"))
	}
	return 0, 0
}

func strToPort(p string) uint16 {
	if p == "" {
		return 0
	}
	pInt, err := strconv.Atoi(p)
	if err != nil {
		return 0
	}
	if pInt <= 0 || pInt > 65535 {
		return 0
	}
	return uint16(pInt)
}
