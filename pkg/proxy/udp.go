package proxy

import (
	"io"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
)

type UDPClient struct {
	dst *net.UDPAddr
	src *net.UDPAddr

	srcSock *net.UDPConn
	dstSock *net.UDPConn
}

func newUDPClient(src, dst *net.UDPAddr) *UDPClient {
	return &UDPClient{
		src: src,
		dst: dst,
	}
}

func (c *UDPClient) Start() error {
	var err error

	c.srcSock, err = net.DialUDP("udp", nil, c.src)
	if err != nil {
		return nil
	}

	c.dstSock, err = net.DialUDP("udp", nil, c.dst)
	if err != nil {
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		transfer(c.dstSock, c.srcSock, &wg)
	}()

	go func() {
		transfer(c.srcSock, c.dstSock, &wg)
	}()

	wg.Wait()
	return nil
}

func transfer(src, dst *net.UDPConn, wg *sync.WaitGroup) {
	defer wg.Done()
	n, err := io.Copy(dst, src)
	if err != nil {
		logrus.WithError(err).Warnf("Failed to transfer from %v to %v", src.RemoteAddr(), dst.RemoteAddr())
	}
	logrus.Infof("Transmitted %v bytes from %v to %v", n, src.RemoteAddr(), dst.RemoteAddr())
}
