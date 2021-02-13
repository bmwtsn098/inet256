package inet256cmd

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/d/celltracker"
	"github.com/brendoncarroll/go-p2p/s/noiseswarm"
	"github.com/brendoncarroll/go-p2p/s/quicswarm"
	"github.com/brendoncarroll/go-p2p/s/udpswarm"
	"github.com/brendoncarroll/go-p2p/s/upnpswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type PeerSpec struct {
	ID    p2p.PeerID `yaml:"id"`
	Addrs []string   `yaml:"addrs"`
}

type TransportSpec struct {
	UDP      *UDPTransportSpec      `yaml:"udp,omitempty"`
	QUIC     *QUICTransportSpec     `yaml:"quic,omitempty"`
	Ethernet *EthernetTransportSpec `yaml:"ethernet,omitempty"`

	NATUPnP bool `yaml:"nat_upnp"`
}

type UDPTransportSpec string
type QUICTransportSpec string
type EthernetTransportSpec string

type DiscoverySpec struct {
	CellTracker *CellTrackerSpec `yaml:"cell_tracker"`
}
type CellTrackerSpec = string

type Config struct {
	PrivateKeyPath string          `yaml:"private_key_path"`
	APIAddr        string          `yaml:"api_addr"`
	Networks       []string        `yaml:"networks"`
	Transports     []TransportSpec `yaml:"transports"`
	Peers          []PeerSpec      `yaml:"peers"`

	Discovery []DiscoverySpec `yaml:"discovery"`
}

func (c Config) GetAPIAddr() string {
	if c.APIAddr == "" {
		return defaultAPIAddr
	}
	return c.APIAddr
}

func MakeNodeParams(configPath string, c Config) (*inet256.Params, error) {
	// private key
	keyPath := c.PrivateKeyPath
	if strings.HasPrefix(c.PrivateKeyPath, "./") {
		keyPath = filepath.Join(filepath.Dir(configPath), c.PrivateKeyPath)
	}
	keyPEMData, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	privateKey, err := inet256.ParsePrivateKeyPEM(keyPEMData)
	if err != nil {
		return nil, err
	}
	// transports
	swarms := map[string]p2p.SecureSwarm{}
	for _, tspec := range c.Transports {
		sw, swname, err := makeTransport(tspec, privateKey)
		if err != nil {
			return nil, err
		}
		if tspec.NATUPnP {
			sw = upnpswarm.WrapSwarm(sw)
		}
		secSw, ok := sw.(p2p.SecureSwarm)
		if !ok {
			secSw = noiseswarm.New(sw, privateKey)
			swname = "noise+" + swname
		}
		swarms[swname] = secSw
	}
	// peers
	peers := inet256.NewPeerStore()
	for _, pspec := range c.Peers {
		peers.Add(pspec.ID)
		peers.SetAddrs(pspec.ID, pspec.Addrs)
	}
	// networks
	nspecs := []inet256.NetworkSpec{}
	for _, netName := range c.Networks {
		nspec, exists := networkNames[netName]
		if !exists {
			return nil, errors.Errorf("no network: %s", netName)
		}
		nspecs = append(nspecs, nspec)
	}

	params := &inet256.Params{
		PrivateKey: privateKey,
		Swarms:     swarms,
		Networks:   nspecs,
		Peers:      peers,
	}
	return params, nil
}

func makeTransport(spec TransportSpec, privKey p2p.PrivateKey) (p2p.Swarm, string, error) {
	switch {
	case spec.UDP != nil:
		s, err := udpswarm.New(string(*spec.UDP))
		if err != nil {
			return nil, "", err
		}
		return s, "udp", nil
	case spec.QUIC != nil:
		s, err := quicswarm.New(string(*spec.QUIC), privKey)
		if err != nil {
			return nil, "", err
		}
		return s, "quic", err
	case spec.Ethernet != nil:
		return nil, "", errors.Errorf("ethernet transport not implemented")
	default:
		return nil, "", errors.Errorf("empty transport spec")
	}
}

func makeDiscoveryService(spec DiscoverySpec) (p2p.DiscoveryService, error) {
	switch {
	case spec.CellTracker != nil:
		return celltracker.NewClient(*spec.CellTracker)
	default:
		return nil, errors.Errorf("empty discovery spec")
	}
}

func DefaultConfig() Config {
	return Config{
		Networks: []string{"onehop"},
		APIAddr:  defaultAPIAddr,
		Transports: []TransportSpec{
			{
				UDP: (*UDPTransportSpec)(strPtr("0.0.0.0")),
			},
		},
	}
}

func LoadConfig(p string) (*Config, error) {
	data, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}
	c := &Config{}
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}
	return c, nil
}

func setupDiscovery(spec DiscoverySpec) (p2p.DiscoveryService, error) {
	switch {
	case spec.CellTracker != nil:
		return celltracker.NewClient(*spec.CellTracker)
	default:
		return nil, errors.Errorf("empty spec")
	}
}

func strPtr(x string) *string {
	return &x
}
