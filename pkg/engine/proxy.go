package engine

import (
	"fmt"
	"time"

	"github.com/mazdakn/uproxy/pkg/packet"
	"golang.org/x/sys/unix"
)

type proxyDevice struct {
	connections *Connections
}

func newProxy() *proxyDevice {
	return &proxyDevice{
		connections: newConnections(),
	}
}

func (p *proxyDevice) Start() error {
	return nil
}

func (p *proxyDevice) Stop() error {
	// TODO: teardown
	return nil
}

func (p *proxyDevice) Name() string {
	return "proxy"
}

func (p *proxyDevice) Read(_ *packet.Packet, _ time.Time) (int, error) {
	panic("proxy device should never be read")
}

func (p *proxyDevice) Write(pkt *packet.Packet, _ time.Time) (int, error) {
	if !protocolSupported(pkt.Protocol()) {
		return 0, fmt.Errorf("protocol %v not supported", packet.ProtoToString(pkt.Protocol()))
	}
	conn := p.connections.Lookup(pkt)
	// Already established connection
	if conn != nil {
		conn.lastActive = time.Now()
		return conn.Conn.Write(pkt.Payload())
	}
	// new connection

	/*if pkt.Protocol() == unix.IPPROTO_UDP {
		udpClient := newUDPClient(
			fmt.Sprintf("%v:%v", pkt.SrcAddr(), pkt.SrcPort()),
			fmt.Sprintf("%v:%v", pkt.DstAddr(), pkt.DstPort()),
		)
	}*/

	return 0, fmt.Errorf("failed to handle packet")
}

func protocolSupported(proto byte) bool {
	if proto == unix.IPPROTO_TCP || proto == unix.IPPROTO_UDP {
		return true
	}
	return false
}
