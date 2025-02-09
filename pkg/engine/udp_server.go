package engine

import (
	"fmt"
	"net"
	"time"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/packet"
)

const (
	queueCapacity = 16
)

type udpServer struct {
	addr  string
	conn  *net.UDPConn
	index uint8

	ingress, egress chan *packet.Packet
}

func newUDPServer(conf *config.Config, index uint8) *udpServer {
	return &udpServer{
		addr:    conf.Address,
		index:   index,
		ingress: make(chan *packet.Packet, queueCapacity),
		egress:  make(chan *packet.Packet, queueCapacity),
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

func (s *udpServer) Stop() error {
	// TODO: teardown listening socket
	return nil
}

func (s *udpServer) Name() string {
	return fmt.Sprintf("udp://%v", s.addr)
}

func (t udpServer) IngressChan() chan<- *packet.Packet {
	return t.ingress
}

func (t udpServer) EgressChan() <-chan *packet.Packet {
	return t.egress
}

func (s *udpServer) Read(pkt *packet.Packet, deadline time.Time) (int, error) {
	err := s.conn.SetReadDeadline(deadline)
	if err != nil {
		return 0, err
	}
	// TODO: check ignored udp address to verify the endpoint
	n, addr, err := s.conn.ReadFrom(pkt.Bytes)
	pkt.Meta.Origin = addr
	pkt.Meta.SrcIndex = s.index
	return n, err
}

func (s *udpServer) Write(pkt *packet.Packet, deadline time.Time) (int, error) {
	err := s.conn.SetWriteDeadline(deadline)
	if err != nil {
		return 0, err
	}
	endpoint := pkt.Meta.Endpoint
	if endpoint == nil {
		return 0, fmt.Errorf("endpoint is not set")
	}
	return s.conn.WriteToUDP(pkt.Bytes, endpoint)
}
