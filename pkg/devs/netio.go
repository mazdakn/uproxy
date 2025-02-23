package devs

import "github.com/mazdakn/uproxy/pkg/packet"

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

	Ingress() chan<- *packet.Packet
	Egress() <-chan *packet.Packet
}
