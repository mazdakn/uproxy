package packet

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

type Packet struct {
	bytes []byte
	ipv6  bool
}

func New(bytes []byte) *Packet {
	return &Packet{
		bytes: bytes,
	}
}

func (p *Packet) Parse() error {
	// At least 20 bytes (IPv4 header length) is needed
	if len(p.bytes) < 20 {
		return fmt.Errorf("Short packet length=%v", len(p.bytes))
	}
	p.ipv6 = p.Version() == 6
	if p.ipv6 && len(p.bytes) < 40 {
		return fmt.Errorf("Short ipv6 packet length=%v", len(p.bytes))
	}
	return nil
}

func (p Packet) Len() int {
	return len(p.bytes)
}

func (p Packet) Version() uint8 {
	return p.bytes[0] >> 4
}

func (p Packet) SrcAddr() net.IP {
	if p.ipv6 {
		return p.bytes[8:24]
	}
	return p.bytes[12:16]
}

func (p Packet) DstAddr() net.IP {
	if p.ipv6 {
		return p.bytes[24:40]
	}
	return p.bytes[16:20]
}

func (p Packet) Protocol() byte {
	if p.ipv6 {
		return p.bytes[6]
	}
	return p.bytes[9]
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
