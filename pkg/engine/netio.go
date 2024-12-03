package engine

import (
	"time"

	"github.com/mazdakn/uproxy/pkg/packet"
)

type NetIO interface {
	Start() error

	Name() string
	Channel() *chan *packet.Packet

	Read(*packet.Packet, time.Time) (int, error)
	Write(*packet.Packet, time.Time) (int, error)
}
