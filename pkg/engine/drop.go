package engine

import (
	"net"
	"time"
)

type dropDevice struct {
	counter int64
}

func newDrop() *dropDevice {
	return &dropDevice{}
}

func (d dropDevice) Start() error {
	return nil
}

func (d dropDevice) Name() string {
	return "drop"
}

func (d dropDevice) Read(_ []byte, _ time.Time) (int, error) {
	panic("drop device should never be read")
}

func (d *dropDevice) Write(_ []byte, _ *net.UDPAddr, _ time.Time) (int, error) {
	d.counter++
	return 0, nil
}
