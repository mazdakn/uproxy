package engine

import (
	"net"
	"strconv"
	"strings"

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

	Action   NetIO
	Endpoint *net.UDPAddr
}

func (p Policy) Match(pkt *Packet) bool {
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
