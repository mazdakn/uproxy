package tun

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
