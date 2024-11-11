package udp

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/sirupsen/logrus"
)

type TunnelUDP struct {
	addr   string
	conn   *net.UDPConn
	writeC chan net.Buffers
}

func New(conf *config.Config) *TunnelUDP {
	return &TunnelUDP{
		addr:   conf.Address,
		writeC: make(chan net.Buffers),
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
	logrus.Infof("Started listening on %v", t.conn.LocalAddr())

	return nil
}

func (t *TunnelUDP) Name() string {
	return fmt.Sprintf("udp://%v", t.addr)
}

func (t TunnelUDP) Backend() io.ReadWriter {
	return t.conn
}

func (t *TunnelUDP) SetReadDeadline(deadline time.Time) error {
	err := t.conn.SetReadDeadline(deadline)
	if err != nil {
		return nil
	}
	return nil
}

func (t *TunnelUDP) WriteC() chan net.Buffers {
	return t.writeC
}
