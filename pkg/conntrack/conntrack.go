package conntrack

import (
	"time"

	"github.com/mazdakn/uproxy/pkg/devs"
	"github.com/mazdakn/uproxy/pkg/packet"
)

type Connection struct {
	InDev devs.NetIO

	OutDev     devs.NetIO
	lastActive time.Time // TODO: change to monotonic time
}

type ConnTable struct {
	conns map[string]Connection
}

func New() *ConnTable {
	return &ConnTable{
		conns: make(map[string]Connection),
	}
}

func (c *ConnTable) Add(pkt *packet.Packet) {
	c.conns[pkt.Tuple()] = Connection{
		lastActive: time.Now(),
	}
}

func (c *ConnTable) Lookup(pkt *packet.Packet) (*Connection, bool) {
	conn, exists := c.conns[pkt.Tuple()]
	return &conn, exists
}

func (c *ConnTable) Delete(pkt *packet.Packet) {
	delete(c.conns, pkt.Tuple())
}
