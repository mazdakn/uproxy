package proxy

import (
	"time"

	"github.com/mazdakn/uproxy/pkg/packet"
)

type Proxy struct {
	writeC chan *packet.Packet
}

func New() *Proxy {
	return &Proxy{
		writeC: make(chan *packet.Packet, 16),
	}
}

func (p Proxy) Start() error {
	return nil
}

func (p Proxy) Name() string {
	return "proxy"
}

func (p Proxy) WriteC() *chan *packet.Packet {
	return &p.writeC
}

func (p *Proxy) Read(pkt *packet.Packet, deadline time.Time) (int, error) {
	return 0, nil
}

func (p *Proxy) Write(pkt *packet.Packet, deadline time.Time) (int, error) {
	return 0, nil
}
