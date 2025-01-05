package engine

import (
	"net"
	"time"

	"github.com/mazdakn/uproxy/pkg/packet"
)

type Connection struct {
	//InDev   NetIO
	SrcAddr net.IP
	DstAddr net.IP

	Proto   byte
	SrcPort uint16
	DstPort uint16

	//OutDev   NetIO
	//Endpoint *net.UDPAddr
	Conn       net.Conn
	lastActive time.Time // TODO: change to monotonic time
}

func (c Connection) Lookup(pkt *packet.Packet) bool {
	/*if c.InDev.Name() != dev.Name() {
		return false
	}*/
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

type Connections struct {
	connections []Connection
}

func newConnections() *Connections {
	return &Connections{}
}

func (c Connections) Lookup(pkt *packet.Packet) *Connection {
	for _, conn := range c.connections {
		if conn.Lookup(pkt) {
			return &conn
		}
	}
	return nil
}

func (c *Connections) Add(pkt *packet.Packet, conn net.Conn) {
	c.connections = append(c.connections, Connection{
		SrcAddr:    pkt.SrcAddr(),
		DstAddr:    pkt.DstAddr(),
		SrcPort:    pkt.SrcPort(),
		DstPort:    pkt.DstPort(),
		Proto:      pkt.Protocol(),
		Conn:       conn,
		lastActive: time.Now(),
	})
}
