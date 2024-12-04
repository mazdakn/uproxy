package engine

import (
	"net"
)

type Connection struct {
	SrcAddr *net.IP
	DstAddr *net.IP

	Proto   byte
	SrcPort uint16
	DstPort uint16

	Device   NetIO
	Endpoint *net.UDPAddr
	Conn     *net.Conn
}

func (c Connection) Lookup(pkt *Packet) bool {
	if c.SrcAddr != nil && !c.SrcAddr.Equal(pkt.SrcAddr()) {
		return false
	}
	if c.DstAddr != nil && !c.DstAddr.Equal(pkt.DstAddr()) {
		return false
	}
	if c.Proto != 0 && c.Proto != pkt.Protocol() {
		return false
	}
	if c.SrcPort != 0 && c.SrcPort != pkt.SrcPort() {
		return false
	}
	if c.DstPort != 0 && c.DstPort != pkt.DstPort() {
		return false
	}
	return true
}
