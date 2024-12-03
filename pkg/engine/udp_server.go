package engine

import (
	"fmt"
	"net"
	"time"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/packet"
)

type udpServer struct {
	addr    string
	conn    *net.UDPConn
	channel chan *packet.Packet
}

func newUDPServer(conf *config.Config) *udpServer {
	return &udpServer{
		addr:    conf.Address,
		channel: make(chan *packet.Packet, 16),
	}
}

func (s *udpServer) Start() error {
	addr, err := net.ResolveUDPAddr("udp", s.addr)
	if err != nil {
		return fmt.Errorf("Invalid address. err: %w", err)
	}

	s.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start udp listener for %v. err: %w", addr, err)
	}
	return nil
}

func (s *udpServer) Name() string {
	return fmt.Sprintf("udp://%v", s.addr)
}

func (s *udpServer) Channel() *chan *packet.Packet {
	return &s.channel
}

func (s *udpServer) Read(pkt *packet.Packet, deadline time.Time) (int, error) {
	err := s.conn.SetReadDeadline(deadline)
	if err != nil {
		return 0, err
	}
	// TODO: check ignored udp address to verify the endpoint
	n, _, err := s.conn.ReadFrom(pkt.Bytes)
	return n, err
}

func (s *udpServer) Write(pkt *packet.Packet, deadline time.Time) (int, error) {
	err := s.conn.SetWriteDeadline(deadline)
	if err != nil {
		return 0, err
	}
	return s.conn.WriteToUDP(pkt.Bytes, pkt.Endpoint)
}
