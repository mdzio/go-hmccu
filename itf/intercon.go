package itf

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"github.com/mdzio/go-logging"
)

const (
	// default CCU RPC path
	rpcPath = "/RPC2"
)

var iLog = logging.Get("itf-intercon")

// Type is the type of a CCU interface (BidCos-RF, HmIP-RF, ...).
type Type int

// Predefined CCU interface types.
const (
	// CCU1 or CCU2/3 with HMW-LGW
	BidCosWired Type = iota
	// CCU1/2/3, RaspberryMatic with RF module or HM-LGW
	BidCosRF
	// CCU1
	System
	// CCU2/3, RaspberryMatic with RF module
	HmIPRF
	// CCU2/3, RaspberryMatic with RF module
	VirtualDevices
)

var (
	typeStr = []string{
		BidCosWired:    "BidCosWired",
		BidCosRF:       "BidCosRF",
		System:         "System",
		HmIPRF:         "HmIPRF",
		VirtualDevices: "VirtualDevices",
	}
	errInvalidItfType = errors.New("Invalid interface type identifier (expected: BidCosWired, BidCosRF, System, HmIPRF, VirtualDevices)")
	errMissingItfType = errors.New("At least one interface type must be specified")
)

// String implements the Stringer interface.
func (t Type) String() string {
	return typeStr[t]
}

// Set implements flag.Value interface.
func (t *Type) Set(value string) error {
	for idx, str := range typeStr {
		if strings.EqualFold(value, str) {
			*t = Type(idx)
			return nil
		}
	}
	return errInvalidItfType
}

// MarshalText implements TextUnmarshaler (for e.g. JSON encoding). For the
// method to be found by the JSON encoder, use a value receiver.
func (t Type) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText implements TextMarshaler (for e.g. JSON decoding).
func (t *Type) UnmarshalText(text []byte) error {
	return t.Set(string(text))
}

// Types is a list of CCU interface types.
type Types []Type

func (it *Types) String() string {
	s := make([]string, len(*it))
	for i, e := range *it {
		s[i] = e.String()
	}
	return strings.Join(s, ",")
}

// Set implements flag.Value interface.
func (it *Types) Set(value string) error {
	*it = nil
	for _, e := range strings.Split(value, ",") {
		if e == "" {
			continue
		}
		var t Type
		if err := t.Set(e); err != nil {
			return err
		}
		*it = append(*it, t)
	}
	if len(*it) == 0 {
		return errMissingItfType
	}
	return nil
}

// config holds the configuration of a CCU interface.
type config struct {
	reGaHssID string
	path      string
	port      int
}

var (
	// configs holds the configurations of all CCU interfaces.
	configs = []config{
		BidCosWired:    {"BidCos-Wired", "", 2000},
		BidCosRF:       {"BidCos-RF", "", 2001},
		System:         {"System", "", 2002},
		HmIPRF:         {"HmIP-RF", "", 2010},
		VirtualDevices: {"VirtualDevices", "/groups", 9292},
	}
)

// Interconnector gives access to the CCU data model and current data point
// values.
type Interconnector struct {
	CCUAddr   string
	Types     Types
	IDPrefix  string
	ServerURL string
	Receiver  Receiver

	clients map[string]*RegisteredClient
}

// Start connects to the CCU and starts querying model and values. An additional
// handler for XMLRPC ist registered at the DefaultServeMux.
func (i *Interconnector) Start() {
	// create interface clients
	i.clients = make(map[string]*RegisteredClient)
	for _, itfType := range i.Types {
		cfg := configs[itfType]
		// create XML-RPC client
		regID := i.IDPrefix + cfg.reGaHssID
		addr := "http://" + i.CCUAddr + ":" + strconv.Itoa(cfg.port) + cfg.path
		iLog.Infof("Creating interface client for %s, %s", addr, cfg.reGaHssID)
		cln := &Client{
			Name:   addr,
			Caller: &xmlrpc.Client{Addr: addr},
		}
		itf := &RegisteredClient{
			Client:          cln,
			RegistrationURL: i.ServerURL + rpcPath,
			RegistrationID:  regID,
			ReGaHssID:       cfg.reGaHssID,
		}
		itf.Setup()
		i.clients[regID] = itf
	}

	// HM RPC dispatcher
	dispatcher := NewDispatcher(i)

	// register XML-RPC handler at the HTTP server
	httpHandler := &xmlrpc.Handler{Dispatcher: dispatcher}
	http.Handle(rpcPath, httpHandler)

	// register at the CCU interfaces
	for _, c := range i.clients {
		c.Start()
	}
}

// Stop disconnects from the CCU and releases ressources.
func (i *Interconnector) Stop() {
	// stop interface clients
	for _, itfClient := range i.clients {
		itfClient.Stop()
	}

	// unregister XMLRPC handler
	// A registered handler at the http.ServeMux can not be unregistered. Maybe
	// a switchable handler should be used, which is always be registered and
	// can be temporarily switched to a not-found handler.
}

// Client returns the specified interface client.
func (i *Interconnector) Client(regID string) (*RegisteredClient, error) {
	cln, ok := i.clients[regID]
	if !ok {
		return nil, errors.New("Unknown interface client ID: " + regID)
	}
	return cln, nil
}

func (i *Interconnector) callbackReceived(interfaceID string) {
	itf, ok := i.clients[interfaceID]
	if !ok {
		iLog.Warning("Callback received for unknown interface ID: ", interfaceID)
		return
	}
	itf.CallbackReceived()
}

// Event implements interface hmccu.Receiver.
func (i *Interconnector) Event(interfaceID, address, valueKey string, value interface{}) error {
	i.callbackReceived(interfaceID)

	// discard pong event
	if valueKey == "PONG" && strings.HasPrefix(address, "CENTRAL") {
		iLog.Trace("Discarding PONG event")
		return nil
	}

	// forward
	return i.Receiver.Event(interfaceID, address, valueKey, value)
}

// NewDevices implements interface hmccu.Receiver.
func (i *Interconnector) NewDevices(interfaceID string, devDescriptions []*DeviceDescription) error {
	i.callbackReceived(interfaceID)

	// forward
	return i.Receiver.NewDevices(interfaceID, devDescriptions)
}

// DeleteDevices implements interface hmccu.Receiver.
func (i *Interconnector) DeleteDevices(interfaceID string, addresses []string) error {
	i.callbackReceived(interfaceID)

	// forward
	return i.Receiver.DeleteDevices(interfaceID, addresses)
}

// UpdateDevice implements interface hmccu.Receiver.
func (i *Interconnector) UpdateDevice(interfaceID, address string, hint int) error {
	i.callbackReceived(interfaceID)

	// forward
	return i.Receiver.UpdateDevice(interfaceID, address, hint)
}

// ReplaceDevice implements interface hmccu.Receiver.
func (i *Interconnector) ReplaceDevice(interfaceID, oldDeviceAddress, newDeviceAddress string) error {
	i.callbackReceived(interfaceID)

	// forward
	return i.Receiver.ReplaceDevice(interfaceID, oldDeviceAddress, newDeviceAddress)
}

// ReaddedDevice implements interface hmccu.Receiver.
func (i *Interconnector) ReaddedDevice(interfaceID string, deletedAddresses []string) error {
	i.callbackReceived(interfaceID)

	// forward
	return i.Receiver.ReaddedDevice(interfaceID, deletedAddresses)
}
