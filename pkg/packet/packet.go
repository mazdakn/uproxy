package packet

import (
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/sirupsen/logrus"
)

type Packet struct {
	NextHdr gopacket.LayerType
	Src     net.IP
	Dst     net.IP
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
