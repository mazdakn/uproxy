package packet

import (
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/sirupsen/logrus"
)

type Packet struct {
	bytes   []byte
	ipv6    bool
	Src     net.IP
	Dst     net.IP
	Proto   int
	SrcPort uint16
	DstPort uint16
}

func Parse(data []byte) gopacket.Packet {
	// At least 20 bytes (IPv4 header length) is needed
	if len(data) < 20 {
		logrus.Warnf("Short packet length=%v", len(data))
		return nil
	}
	pkt := gopacket.NewPacket(data, layers.LayerTypeIPv4, gopacket.Default)
	ipLayer := pkt.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		logrus.Warnf("Failed to parse packet")
		return nil
	}
	logrus.Infof("Packet: %v", pkt)
	return pkt
}

func (p *Packet) Parse() error {
	p.ipv6 = p.Version() == 6
	return nil
}

func (p Packet) Len() int {
	return len(p.bytes)
}

func (p Packet) Version() uint8 {
	return p.bytes[0]
}

func (p Packet) SrcAddr() net.IP {
	return p.bytes[12:16]
}

func (p Packet) DstAddr() net.IP {
	return p.bytes[16:20]
}

func (p Packet) Protocol() byte {
	return p.bytes[9:10]

}
