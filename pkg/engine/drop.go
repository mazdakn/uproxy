package engine

import (
	"sync"
	"time"

	"github.com/mazdakn/uproxy/pkg/packet"
)

type dropDevice struct {
	lock    sync.RWMutex
	counter int64
}

func newDrop() *dropDevice {
	return &dropDevice{}
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

func (d *dropDevice) Read(_ *packet.Packet, _ time.Time) (int, error) {
	panic("drop device should never be read")
}

func (d *dropDevice) Write(pkt *packet.Packet, _ time.Time) (int, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.counter++
	return pkt.Len(), nil
}
