package tun

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
	"unsafe"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/packet"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	cloneDevicePath = "/dev/net/tun"
	ifReqSize       = unix.IFNAMSIZ + 64
)

type TunDevice struct {
	conf   *config.Config
	writeC chan net.Buffers
	name   string
	file   *os.File
	mtu    int
}

func New(conf *config.Config) *TunDevice {
	return &TunDevice{
		name: "uproxy",
		mtu:  1400,
	}
}

func (t *TunDevice) Start(ctx context.Context, wg *sync.WaitGroup) (int, error) {
	err := t.create()
	if err != nil {
		return 0, err
	}

	go t.Read(ctx, wg)
	go t.Write(ctx, wg)
	return 2, nil
}

func (t *TunDevice) Read(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	logrus.Infof("Started goroutine reading from %v", t.Name())
	//var err error
	//var buffer net.Buffers
	buffer := make([]byte, t.mtu) // or something else
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("Stopped goroutine reading from %v", t.Name())
			return
		default:
			err := t.file.SetReadDeadline(time.Now().Add(time.Second))
			if err != nil {
				logrus.Errorf("Failed to set read deadline")
			}
			num, err := t.file.Read(buffer)
			if err != nil {
				nerr, ok := err.(net.Error)
				if ok && !nerr.Timeout() {
					logrus.Errorf("failure in reading from %v", t.Name())
				}
			}
			// Nothing recived.
			if num == 0 {
				continue
			}
			logrus.Infof("Received %v bytes from %v.", num, t.Name())
			packet.Parse(buffer[:num])
		}

	}
}

func (t *TunDevice) Write(ctx context.Context, wg *sync.WaitGroup) {
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
			num, err = packets.WriteTo(t.file)
			if err != nil {
				logrus.Errorf("Failed to write to %v", t.Name())
				continue
			}
			logrus.Debugf("Sent %v packets via %v", num, t.Name())
		}
	}
}

func (t *TunDevice) create() error {
	nfd, err := unix.Open(cloneDevicePath, unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		/*f os.IsNotExist(err) {
			return fmt.Errorf("failed", t.name, cloneDevicePath)
		}*/
		return err
	}

	ifr, err := unix.NewIfreq(t.name)
	if err != nil {
		return err
	}
	// IFF_VNET_HDR enables the "tun status hack" via routineHackListener()
	// where a null write will return EINVAL indicating the TUN is up.
	ifr.SetUint16(unix.IFF_TUN | unix.IFF_NO_PI)
	err = unix.IoctlIfreq(nfd, unix.TUNSETIFF, ifr)
	if err != nil {
		return err
	}

	err = unix.SetNonblock(nfd, true)
	if err != nil {
		return err
	}

	t.file = os.NewFile(uintptr(nfd), cloneDevicePath)

	err = t.setMTU()
	if err != nil {
		return err
	}

	return nil
}

func (t *TunDevice) setMTU() error {
	// open datagram socket
	fd, err := unix.Socket(
		unix.AF_INET,
		unix.SOCK_DGRAM|unix.SOCK_CLOEXEC,
		0,
	)
	if err != nil {
		return err
	}

	defer unix.Close(fd)

	// do ioctl call
	var ifr [ifReqSize]byte
	copy(ifr[:], t.name)
	*(*uint32)(unsafe.Pointer(&ifr[unix.IFNAMSIZ])) = uint32(t.mtu)
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(unix.SIOCSIFMTU),
		uintptr(unsafe.Pointer(&ifr[0])),
	)

	if errno != 0 {
		return fmt.Errorf("failed to set MTU of TUN device: %w", errno)
	}

	return nil
}

func (t *TunDevice) MTU() (int, error) {
	// open datagram socket
	fd, err := unix.Socket(
		unix.AF_INET,
		unix.SOCK_DGRAM|unix.SOCK_CLOEXEC,
		0,
	)
	if err != nil {
		return 0, err
	}

	defer unix.Close(fd)

	// do ioctl call

	var ifr [ifReqSize]byte
	copy(ifr[:], t.name)
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(unix.SIOCGIFMTU),
		uintptr(unsafe.Pointer(&ifr[0])),
	)
	if errno != 0 {
		return 0, fmt.Errorf("failed to get MTU of TUN device: %w", errno)
	}

	return int(*(*int32)(unsafe.Pointer(&ifr[unix.IFNAMSIZ]))), nil
}

func (t *TunDevice) WriteChannel() chan<- net.Buffers {
	return t.writeC
}

func (t *TunDevice) Name() string {
	return fmt.Sprintf("tun %v", t.name)
}
