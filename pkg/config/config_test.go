package config

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestDefaults(t *testing.T) {
	RegisterTestingT(t)
	defaultConfig := newWithDefaults()
	Expect(defaultConfig).To(Equal(&Config{
		MaxBufferSize: defautMaxBufferSize,
	}))
}

func TestDefaultCmdLine(t *testing.T) {
	RegisterTestingT(t)
	config := FromCmdline()
	configDefault := newWithDefaults()
	configDefault.Addr = defaultAddr
	Expect(config).To(Equal(configDefault))
}
