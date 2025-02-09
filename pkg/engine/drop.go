package engine

import (
	"time"

	"github.com/mazdakn/uproxy/pkg/packet"
)

type dropDevice struct {
	counter         int64
	ingress, egress chan *packet.Packet
}

func newDrop() *dropDevice {
	return &dropDevice{
		ingress: make(chan *packet.Packet, queueCapacity),
		egress:  make(chan *packet.Packet, queueCapacity),
	}
}

func (d *dropDevice) Start() error {
	return nil
}

func (d *dropDevice) Stop() error {
	return nil
}

func (d *dropDevice) Name() string {
	return "drop"
}

func (d dropDevice) IngressChan() chan<- *packet.Packet {
	return d.ingress
}

func (d dropDevice) EgressChan() <-chan *packet.Packet {
	return d.egress
}

func (d *dropDevice) Read(_ *packet.Packet, _ time.Time) (int, error) {
	panic("drop device should never be read")
}

func (d *dropDevice) Write(pkt *packet.Packet, _ time.Time) (int, error) {
	d.counter++
	return pkt.Len(), nil
}
