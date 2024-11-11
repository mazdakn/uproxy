package udp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/packet"
	"github.com/sirupsen/logrus"
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

func (t *TunnelUDP) Start(ctx context.Context, wg *sync.WaitGroup) error {
	addr, err := net.ResolveUDPAddr("udp", t.addr)
	if err != nil {
		return fmt.Errorf("Invalid address. err: %w", err)
	}

	t.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start udp listener for %v. err: %w", addr, err)
	}
	logrus.Infof("Started listening on %v", t.conn.LocalAddr())

	return nil
}

func (t *TunnelUDP) Name() string {
	return fmt.Sprintf("udp://%v", t.addr)
}

func (t *TunnelUDP) WriteC() chan *packet.Packet {
	return t.writeC
}

func (t *TunnelUDP) Read(pkt *packet.Packet) (int, error) {
	// TODO: check ignored udp address to verify the endpoint
	n, _, err := t.conn.ReadFrom(pkt.Bytes)
	return n, err
}

func (t *TunnelUDP) SetReadDeadline(deadline time.Time) error {
	return t.conn.SetReadDeadline(deadline)
}

func (t *TunnelUDP) Write(pkt *packet.Packet) (int, error) {
	return t.conn.WriteToUDP(pkt.Bytes, pkt.Endpoint)
}

func (t *TunnelUDP) SetWriteDeadline(deadline time.Time) error {
	return t.conn.SetReadDeadline(deadline)
}
