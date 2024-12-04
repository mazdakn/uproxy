package engine

import (
	"net"
	"time"
)

type NetIO interface {
	Start() error

	Name() string

	Read([]byte, time.Time) (int, error)
	Write([]byte, *net.UDPAddr, time.Time) (int, error)
}
