package config

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestDefaults(t *testing.T) {
	RegisterTestingT(t)
	defaultConfig := Config{}
	ApplyDefaults(&defaultConfig)
	Expect(defaultConfig).To(Equal(&Config{
		MaxBufferSize: defautMaxBufferSize,
	}))
}

func TestDefaultCmdLine(t *testing.T) {
	RegisterTestingT(t)
	cliConfig, err := FromCmdline()
	if err != nil {
		t.Fail()
	}
	defaultConfig := Config{}
	ApplyDefaults(&defaultConfig)
	Expect(cliConfig).To(Equal(defaultConfig))
}
