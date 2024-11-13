package packet

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

type Packet struct {
	Bytes    []byte
	Size     int
	ipv6     bool
	Endpoint *net.UDPAddr
}

func New(MaxBufferSize int) *Packet {
	return &Packet{
		Bytes: make([]byte, MaxBufferSize),
	}
}

func (p *Packet) Parse(size int) error {
	p.Bytes = p.Bytes[:size]
	// At least 20 bytes (IPv4 header length) is needed
	if len(p.Bytes) < 20 {
		return fmt.Errorf("Short packet length=%v", len(p.Bytes))
	}
	p.ipv6 = p.Version() == 6
	if p.ipv6 && len(p.Bytes) < 40 {
		return fmt.Errorf("Short ipv6 packet length=%v", len(p.Bytes))
	}
	return nil
}

func (p Packet) Len() int {
	return len(p.Bytes)
}

func (p Packet) Version() uint8 {
	return p.Bytes[0] >> 4
}

func (p Packet) SrcAddr() net.IP {
	if p.ipv6 {
		return p.Bytes[8:24]
	}
	return p.Bytes[12:16]
}

func (p Packet) DstAddr() net.IP {
	if p.ipv6 {
		return p.Bytes[24:40]
	}
	return p.Bytes[16:20]
}

func (p Packet) Protocol() byte {
	if p.ipv6 {
		return p.Bytes[6]
	}
	return p.Bytes[9]
}

func (p Packet) String() string {
	var proto string
	switch p.Protocol() {
	case unix.IPPROTO_UDP:
		proto = "udp"
	case unix.IPPROTO_TCP:
		proto = "tcp"
	case unix.IPPROTO_ICMP:
		proto = "icmp"
	case unix.IPPROTO_ICMPV6:
		proto = "icmp6"
	default:
		proto = "xxx"
	}
	return fmt.Sprintf("%v(%v -> %v) len: %v", proto, p.SrcAddr().String(), p.DstAddr().String(), p.Len())
}
