package engine

import (
	"time"

	"github.com/mazdakn/uproxy/pkg/packet"
)

const (
	NetIO_Drop = iota
	NetIO_UDPServer
	NetIO_Local
	NetIO_Proxy

	NetIO_Max
	NetIO_Error
)

type NetIO interface {
	Start() error
	Stop() error

	Name() string

	Read(*packet.Packet, time.Time) (int, error)
	Write(*packet.Packet, time.Time) (int, error)
}
