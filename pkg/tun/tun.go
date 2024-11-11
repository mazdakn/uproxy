package tun

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"
	"unsafe"

	"github.com/mazdakn/uproxy/pkg/config"
	"golang.org/x/sys/unix"
)

const (
	cloneDevicePath = "/dev/net/tun"
	ifReqSize       = unix.IFNAMSIZ + 64
)

type TunDevice struct {
	name   string
	file   *os.File
	writeC chan net.Buffers
	mtu    int
}

func New(conf *config.Config) *TunDevice {
	return &TunDevice{
		name:   "uproxy",
		mtu:    1400,
		writeC: make(chan net.Buffers),
	}
}

func (t *TunDevice) Start() error {
	err := t.create()
	if err != nil {
		return err
	}
	return nil
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

func (t *TunDevice) Name() string {
	return fmt.Sprintf("tun %v", t.name)
}

func (t TunDevice) Backend() io.ReadWriter {
	return t.file
}

func (t *TunDevice) SetReadDeadline(deadline time.Time) error {
	err := t.file.SetReadDeadline(deadline)
	if err != nil {
		return err
	}
	return nil
}

func (t TunDevice) WriteC() chan net.Buffers {
	return t.writeC
}
