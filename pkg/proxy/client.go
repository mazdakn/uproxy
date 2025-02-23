package proxy

import "github.com/mazdakn/uproxy/pkg/packet"

func newClient() *client {
	return &client{
		ingress: make(chan *packet.Packet, queueCapacity),
		egress:  make(chan *packet.Packet, queueCapacity),
	}
}

type client struct {
	ingress, egress chan *packet.Packet
}

func (c *client) Start() error {
	return nil
}

func (c *client) Stop() error {
	return nil
}

func (c *client) Name() string {
	return "client"
}

func (c client) Ingress() chan<- *packet.Packet {
	return c.ingress
}

func (c client) Egress() <-chan *packet.Packet {
	return c.egress
}
