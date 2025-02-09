package tun

import "net"

type Option func(*TunDevice)

func WithName(name string) Option {
	return func(t *TunDevice) {
		t.name = name
	}
}

func WithMTU(mtu int) Option {
	return func(t *TunDevice) {
		t.mtu = mtu
	}
}

func WithAddress(addr string) Option {
	return func(t *TunDevice) {
		t.address = addr
	}
}

func WithEgress(udpConn *net.UDPConn) Option {
	return func(t *TunDevice) {
		t.udpConn = udpConn
	}
}

func WithBufferSize(size int) Option {
	return func(t *TunDevice) {
		t.bufferSize = size
	}
}
