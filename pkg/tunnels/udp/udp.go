package udp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/packet"
	"github.com/sirupsen/logrus"
)

type TunnelUDP struct {
	conf       *config.Config
	addr       string
	conn       *net.UDPConn
	writeC     chan net.Buffers
	bufferSize int
}

func New(conf *config.Config) *TunnelUDP {
	return &TunnelUDP{
		addr:       conf.Address,
		bufferSize: conf.MaxBufferSize,
	}
}

func (t *TunnelUDP) Start(ctx context.Context, wg *sync.WaitGroup) (int, error) {
	addr, err := net.ResolveUDPAddr("udp", t.addr)
	if err != nil {
		return 0, fmt.Errorf("Invalid address. err: %w", err)
	}

	t.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return 0, fmt.Errorf("failed to start udp listener for %v. err: %w", addr, err)
	}
	logrus.Infof("Started listening on %v", t.conn.LocalAddr())

	go t.Read(ctx, wg)
	go t.Write(ctx, wg)
	return 2, nil
}

func (t *TunnelUDP) Read(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	logrus.Infof("Started goroutine reading from %v", t.Name())
	var err error
	buffer := make([]byte, t.bufferSize)
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("Stoped goroutine reading from %v", t.Name())
			return
		default:
			err = t.conn.SetReadDeadline(time.Now().Add(time.Second))
			if err != nil {
				logrus.Errorf("Failed to set read deadline")
			}
			num, addr, err := t.conn.ReadFrom(buffer)
			if err != nil {
				nerr, ok := err.(net.Error)
				if ok && !nerr.Timeout() {
					logrus.Errorf("failure in reading from %v", addr)
				}
			}
			// Nothing recived.
			if num == 0 {
				continue
			}
			logrus.Infof("Received %v bytes from %v.", num, addr)
			packet.Parse(buffer[:num])
		}
	}
}

func (t *TunnelUDP) Write(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	logrus.Infof("Started goroutine writing to %v", t.Name())
	var err error
	var num int64
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("Stoped goroutine writing to %v", t.Name())
			return
		case packets := <-t.writeC:
			num, err = packets.WriteTo(t.conn)
			if err != nil {
				logrus.Errorf("Failed to write to %v", t.Name())
				continue
			}
			logrus.Debugf("Sent %v packets via %v", num, t.Name())
		}
	}
}

func (t *TunnelUDP) WriteChannel() chan<- net.Buffers {
	return t.writeC
}

func (t *TunnelUDP) Name() string {
	return fmt.Sprintf("udp://%v", t.addr)
}

func handlePacket(buffer []byte) error {
	pkt := packet.Parse(buffer)
	if pkt == nil {
		return fmt.Errorf("Failed to parse packet")
	}
	return nil
}
