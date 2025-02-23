package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mazdakn/uproxy/pkg/conntrack"
	"github.com/mazdakn/uproxy/pkg/packet"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	queueCapacity  = 16
	bufferCapacity = 1600
)

type Proxy struct {
	lAddr   string
	lConn   *net.UDPConn
	ingress chan *packet.Packet
	//connections map[string]chan *packet.Packet
	connections *conntrack.ConnTable
}

func New(addr string) *Proxy {
	return &Proxy{
		lAddr:       addr,
		ingress:     make(chan *packet.Packet, queueCapacity),
		connections: conntrack.New(),
	}
}

func (p *Proxy) Start() error {
	addr, err := net.ResolveUDPAddr("udp", p.lAddr)
	if err != nil {
		return fmt.Errorf("Invalid address. err: %w", err)
	}
	p.lConn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start udp listener for %v. err: %w", addr, err)
	}

	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(2)

	go p.readPackets(ctx, &wg, time.Second*1)
	go p.handlePackets(ctx, &wg)

	wg.Wait()
	return nil
}

func (p *Proxy) readPackets(ctx context.Context, wg *sync.WaitGroup, deadline time.Duration) {
	logrus.Infof("Started routine reading packets")
	defer wg.Done()
	for {
		if ctx.Err() != nil {
			logrus.Info("Context Cancelled. Stopping goroutine reading packets")
			return
		}

		err := p.lConn.SetReadDeadline(time.Now().Add(deadline))
		if err != nil {
			logrus.WithError(err).Error("Failed to set reading deadline")
			continue
		}
		pkt := packet.New(bufferCapacity)
		n, addr, err := p.lConn.ReadFromUDP(pkt.Bytes)
		if err != nil {
			nerr, ok := err.(net.Error)
			if ok && !nerr.Timeout() {
				logrus.WithError(err).Error("failure in reading from packet")
				continue
			}
		}
		if n == 0 {
			logrus.Info("Received empty packet")
			continue
		}
		pkt.Size = n
		pkt.Meta.Origin = addr
		pkt.Meta.SrvConn = p.lConn
		p.ingress <- pkt
	}
}

func (p *Proxy) handlePackets(ctx context.Context, wg *sync.WaitGroup) {
	logrus.Infof("Started routine processing packets")
	defer wg.Done()
	var pkts []*packet.Packet
	for {
		if ctx.Err() != nil {
			logrus.Info("Context Cancelled. Stopping goroutine handling packets")
			return
		}

		// Drain the channel
		pkts = nil
		for pkt := range p.ingress {
			pkts = append(pkts, pkt)
		}

		for _, pkt := range pkts {
			if err := pkt.Parse(); err != nil {
				logrus.WithError(err).Error("Failed to parse packet")
				continue
			}
			logrus.Infof("Packet : %v", pkt)

			if !supportedProtocols(pkt.Protocol()) {
				logrus.Warnf("Unsupportd protocol %v", pkt.Protocol())
				continue
			}

			conn, exists := p.connections.Lookup(pkt)
			if exists && conn.OutDev != nil {
				conn.OutDev.Ingress() <- pkt
				continue
			}

			// This is a new connection
			wg.Add(1)
			ingress := make(chan *packet.Packet, queueCapacity)
			go p.handleClient(ctx, wg, pkt, &ingress)
			p.connections.Add(pkt)
			ingress <- pkt
		}
	}
}

func (p *Proxy) handleClient(
	ctx context.Context,
	wg *sync.WaitGroup,
	initPkt *packet.Packet,
	ingress *chan *packet.Packet,
) {
	logrus.Infof("starting a new UDP client %v", initPkt.Tuple())
	defer func() {
		// TODO: might need to lock
		p.connections.Delete(initPkt)
		wg.Done()
	}()
	var pkts []*packet.Packet
	for {
		if ctx.Err() != nil {
			logrus.Info("Context Cancelled. Stopping UDP client")
			return
		}

		// Drain the channel
		pkts = nil
		for pkt := range *ingress {
			pkts = append(pkts, pkt)
		}

		var remoteConn net.Conn
		var err error
		for _, pkt := range pkts {
			if remoteConn == nil {
				proto := protoToStr(pkt.Protocol())
				remoteAddr := fmt.Sprintf("%v:%v", pkt.DstAddr(), pkt.DstPort())
				remoteConn, err = net.Dial(proto, remoteAddr)
				if err != nil {
					logrus.WithError(err).Errorf("Failed to connect to remote %v", remoteAddr)
					return
				}
				logrus.Infof("Connected to remote %v", remoteAddr)
			}

			payload := pkt.Payload()

		}
	}
}

func supportedProtocols(proto byte) bool {
	return proto == unix.IPPROTO_UDP || proto == unix.IPPROTO_TCP
}

func protoToStr(proto byte) string {
	switch proto {
	case unix.IPPROTO_UDP:
		return "udp"
	case unix.IPPROTO_TCP:
		return "tcp"
	default:
		panic("unsupported protocol")
	}
}
