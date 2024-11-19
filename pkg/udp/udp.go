package udp

import (
	"fmt"
	"net"
	"time"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/packet"
)

type TunnelUDP struct {
	addr   string
	conn   *net.UDPConn
	writeC chan *packet.Packet
	conf   *config.Config
}

func New(conf *config.Config) *TunnelUDP {
	return &TunnelUDP{
		conf:   conf,
		addr:   conf.Address,
		writeC: make(chan *packet.Packet, 16),
	}
}

func (t *TunnelUDP) Start() error {
	addr, err := net.ResolveUDPAddr("udp", t.addr)
	if err != nil {
		return fmt.Errorf("Invalid address. err: %w", err)
	}

	t.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start udp listener for %v. err: %w", addr, err)
	}
	return nil
}

func (t *TunnelUDP) Name() string {
	return fmt.Sprintf("udp://%v", t.addr)
}

func (t *TunnelUDP) WriteC() *chan *packet.Packet {
	return &t.writeC
}

func (t *TunnelUDP) Read(pkt *packet.Packet, deadline time.Time) (int, error) {
	err := t.conn.SetReadDeadline(deadline)
	if err != nil {
		return 0, err
	}
	// TODO: check ignored udp address to verify the endpoint
	n, _, err := t.conn.ReadFrom(pkt.Bytes)
	return n, err
}

func (t *TunnelUDP) Write(pkt *packet.Packet, deadline time.Time) (int, error) {
	err := t.conn.SetWriteDeadline(deadline)
	if err != nil {
		return 0, err
	}
	return t.conn.WriteToUDP(pkt.Bytes, pkt.Endpoint)
}
