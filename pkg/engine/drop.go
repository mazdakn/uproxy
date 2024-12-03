package engine

import (
	"time"

	"github.com/mazdakn/uproxy/pkg/packet"
)

type dropDevice struct {
	counter int64
	channel chan *packet.Packet
}

func newDrop() *dropDevice {
	return &dropDevice{
		channel: make(chan *packet.Packet, 16),
	}
}

func (d dropDevice) Start() error {
	return nil
}

func (d dropDevice) Name() string {
	return "drop"
}

func (d dropDevice) Channel() *chan *packet.Packet {
	return &d.channel
}

func (d dropDevice) Read(_ *packet.Packet, _ time.Time) (int, error) {
	panic("drop device should never be read")
}

func (d *dropDevice) Write(_ *packet.Packet, _ time.Time) (int, error) {
	d.counter++
	return 0, nil
}
