package tun

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

type TunDevice struct {
	name    string
	file    *os.File
	mtu     int
	address string
	dev     *netlink.Tuntap
}

func New(conf *config.Config) *TunDevice {
	return &TunDevice{
		name:    conf.Tun.Name,
		mtu:     conf.Tun.MTU,
		address: conf.Tun.Address,
	}
}

func (t *TunDevice) Start() error {
	logrus.Infof("Creating tun device %v (address: %v, mtu: %v)", t.name, t.address, t.mtu)
	err := t.create()
	if err != nil {
		return err
	}
	return nil
}

func (t *TunDevice) create() error {
	la := netlink.NewLinkAttrs()
	la.Name = t.name
	la.MTU = t.mtu
	tunDev := &netlink.Tuntap{
		LinkAttrs: la,
		Mode:      netlink.TUNTAP_MODE_TUN,
		Flags:     netlink.TUNTAP_NO_PI | netlink.TUNTAP_MULTI_QUEUE_DEFAULTS,
		Queues:    1,
	}
	err := netlink.LinkAdd(tunDev)
	if err != nil {
		return fmt.Errorf("failed to create tun device - err: %w", err)
	}

	addr, err := netlink.ParseAddr(t.address)
	if err != nil {
		return fmt.Errorf("invaid address %v - err: %w", t.address, err)
	}

	err = netlink.AddrAdd(tunDev, addr)
	if err != nil {
		return fmt.Errorf("failed to set address %v to tun device - err: %w", t.address, err)
	}

	err = netlink.LinkSetMTU(tunDev, t.mtu)
	if err != nil {
		return fmt.Errorf("failed to set tun device mtu to %v - err: %w", t.mtu, err)
	}

	err = netlink.LinkSetUp(tunDev)
	if err != nil {
		return fmt.Errorf("failed to set tun device up - err: %w", err)
	}

	if len(tunDev.Fds) == 0 {
		return fmt.Errorf("no valid queue available for tun device")
	}
	t.dev = tunDev
	t.file = tunDev.Fds[0]

	return nil
}

func (t *TunDevice) Name() string {
	return fmt.Sprintf("tun %v", t.name)
}

func (t TunDevice) Read(pkt []byte, deadline time.Time) (int, error) {
	err := t.file.SetReadDeadline(deadline)
	if err != nil {
		return 0, err
	}
	return t.file.Read(pkt)
}

func (t TunDevice) Write(pkt []byte, _ *net.UDPAddr, deadline time.Time) (int, error) {
	err := t.file.SetWriteDeadline(deadline)
	if err != nil {
		return 0, err
	}
	return t.file.Write(pkt)
}

func (t TunDevice) Stop() error {
	link, err := netlink.LinkByName(t.name)
	if err != nil {
		return fmt.Errorf("failed to find tun device %v - err: %w", t.name, err)
	}
	err = netlink.LinkDel(link)
	if err != nil {
		return fmt.Errorf("failed to delete tun device %v - err: %w", t.name, err)
	}
	return nil
}
