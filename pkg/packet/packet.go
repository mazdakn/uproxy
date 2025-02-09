package packet

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/sys/unix"
)

type Metadata struct {
	// The followings are set when packet is read
	SrcIndex uint8
	Origin   net.Addr
	SrvConn  *net.UDPConn

	// The following is set by policy matcher to endpoint packet should be sent
	Endpoint *net.UDPAddr
}

type Packet struct {
	Bytes []byte
	Size  int
	ipv6  bool
	pkt   gopacket.Packet

	Meta Metadata
}

func New(MaxBufferSize int) *Packet {
	return &Packet{
		Bytes: make([]byte, MaxBufferSize),
	}
}

func (p *Packet) Reset() {
	p.Meta = Metadata{}
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

	if p.ipv6 {
		p.pkt = gopacket.NewPacket(p.Bytes, layers.LayerTypeIPv6, gopacket.Default)
	} else {
		p.pkt = gopacket.NewPacket(p.Bytes, layers.LayerTypeIPv4, gopacket.Default)
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

func (p Packet) SrcPort() uint16 {
	l4Offset := (p.Bytes[0] & 0x0f) * 4
	if p.ipv6 {
		l4Offset = 40
	}
	return binary.BigEndian.Uint16(p.Bytes[l4Offset : l4Offset+2])
}

func (p Packet) DstPort() uint16 {
	l4Offset := (p.Bytes[0] & 0x0f) * 4
	if p.ipv6 {
		l4Offset = 40
	}
	return binary.BigEndian.Uint16(p.Bytes[l4Offset+2 : l4Offset+4])
}

func (p Packet) Payload() []byte {
	start := (p.Bytes[0]<<4)*32 + 8
	return p.Bytes[start:] // only for ipv4 + udp
}

func (p Packet) Routed() bool {
	return p.Meta.Endpoint != nil
}

func (p Packet) String() string {
	p.pkt.Dump()

	switch p.Protocol() {
	case unix.IPPROTO_UDP:
		return fmt.Sprintf("udp(%v:%v -> %v:%v) len: %v",
			p.SrcAddr().String(), p.SrcPort(), p.DstAddr().String(), p.DstPort(), p.Len())
	case unix.IPPROTO_TCP:
		return fmt.Sprintf("tcp(%v:%v -> %v:%v) len: %v",
			p.SrcAddr().String(), p.SrcPort(), p.DstAddr().String(), p.DstPort(), p.Len())
	case unix.IPPROTO_ICMP:
		return fmt.Sprintf("icmp(%v -> %v) len: %v", p.SrcAddr().String(), p.DstAddr().String(), p.Len())
	case unix.IPPROTO_ICMPV6:
		return fmt.Sprintf("icmp6(%v -> %v) len: %v", p.SrcAddr().String(), p.DstAddr().String(), p.Len())
	}
	return "unknown packet"
}

func ProtoToString(proto byte) string {
	switch proto {
	case unix.IPPROTO_TCP:
		return "tcp"
	case unix.IPPROTO_UDP:
		return "udp"
	case unix.IPPROTO_ICMP:
		return "icmp"
	case unix.IPPROTO_ICMPV6:
		return "icmpv6"
	default:
		return "unsupported"
	}
}
