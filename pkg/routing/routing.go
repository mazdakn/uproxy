package routing

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/mazdakn/uproxy/pkg/config"
	"github.com/mazdakn/uproxy/pkg/packet"
	"github.com/sirupsen/logrus"
)

type NetIO interface {
	Start(context.Context, *sync.WaitGroup) error
	Name() string
	WriteC() *chan *packet.Packet

	Read(*packet.Packet, time.Time) (int, error)
	Write(*packet.Packet, time.Time) (int, error)
}

type routeEntry struct {
	Device   NetIO
	Endpoint *net.UDPAddr
}

type RouteTabel struct {
	// TODO: change this to a trie struct with IPNet as keys
	routes  map[string]routeEntry
	devices []NetIO
	conf    *config.Config
}

func New(conf *config.Config) *RouteTabel {
	return &RouteTabel{
		conf:   conf,
		routes: make(map[string]routeEntry),
	}
}

func (t *RouteTabel) ParseRoutes() error {
	for _, r := range t.conf.Routes {
		if len(r.Destinations) == 0 || r.Endpoint == "" {
			logrus.Errorf("Invalid route: %v - Skipping.", r)
			continue
		}
		for _, dest := range r.Destinations {
			_, cidr, err := net.ParseCIDR(dest)
			if err != nil {
				logrus.WithError(err).Errorf("Failed to parse destination %v - Skipping", dest)
				continue
			}
			dev, ep, err := t.selectForwardingDevice(r.Endpoint)
			if err != nil {
				logrus.WithError(err).Errorf("Failed to parse endpoint %v - Skipping", r.Endpoint)
				continue
			}
			epAddr, err := net.ResolveUDPAddr("udp", ep)
			if err != nil {
				logrus.WithError(err).Errorf("Failed to resolve endpoint %v - Skipping", ep)
				continue
			}
			t.routes[cidr.String()] = routeEntry{
				Device:   dev,
				Endpoint: epAddr,
			}
			logrus.WithFields(logrus.Fields{
				"destination": dest,
				"device":      dev.Name(),
				"endpoint":    epAddr,
			}).Debugf("Added route.")
		}
	}

	return nil
}

func (t RouteTabel) selectForwardingDevice(endpoint string) (NetIO, string, error) {
	if endpoint == "local" {
		return t.devices[1], "", nil // using indexes are a hack. Need to be fixed
	}
	if strings.HasPrefix(endpoint, "udp://") {
		return t.devices[0], strings.TrimLeft(endpoint, "udp://"), nil
	}
	return nil, "", fmt.Errorf("failed to parse endpoint %v", endpoint)
}

func (t *RouteTabel) RegisterDevice(dev NetIO) {
	t.devices = append(t.devices, dev)
}

func (t *RouteTabel) Lookup(addr net.IP) *routeEntry {
	logrus.Debugf("Looking up addr %v", addr)
	for cidr, entry := range t.routes {
		_, cidrNet, _ := net.ParseCIDR(cidr) // TODO: fix it
		if cidrNet.Contains(addr) {
			return &entry
		}
	}
	return nil
}
