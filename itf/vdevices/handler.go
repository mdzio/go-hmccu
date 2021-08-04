package vdevices

import (
	"fmt"
	"sync"

	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-lib/conc"
	"github.com/mdzio/go-logging"
)

var log = logging.Get("v-devices")

// EventPublisher publishes value change events.
type EventPublisher interface {
	PublishEvent(address, valueKey string, value interface{})
}

// Synchronizer updates the device lists in the logic layers.
type Synchronizer interface {
	Synchronize()
}

// Handler handles requests from logic layers.
type Handler struct {
	ccuAddr          string
	devices          *Container
	deletionNotifier func(address string)

	servants   map[string]*servant // key: receiverAddress
	mtx        sync.Mutex          // for servants map
	daemonPool conc.DaemonPool     // for background tasks
}

// NewHandler creates a Handler. deletionNotifier is called, when the CCU
// initiates a device deletion.
func NewHandler(ccuAddr string, devices *Container, deletionNotifier func(address string)) *Handler {
	return &Handler{
		ccuAddr:          ccuAddr,
		devices:          devices,
		deletionNotifier: deletionNotifier,
		servants:         make(map[string]*servant),
	}
}

// Close frees resources.
func (h *Handler) Close() {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	for _, s := range h.servants {
		h.daemonPool.Run(func(conc.Context) { s.close() })
	}
	h.servants = make(map[string]*servant)
	h.daemonPool.Close()
}

// Synchronize updates the device lists in the logic layers. Implements
// Synchronizer.
func (h *Handler) Synchronize() {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	for _, s := range h.servants {
		s.command(servantSync{})
	}
}

// PublishEvent distributes an value event to all registered logic layers.
// Implements EventPublisher.
func (h *Handler) PublishEvent(address, valueKey string, value interface{}) {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	log.Tracef("Publishing event: %s, %s, %v", address, valueKey, value)
	for _, s := range h.servants {
		s.command(servantEvent{
			address:  address,
			valueKey: valueKey,
			value:    value,
		})
	}
}

// Init implements DeviceLayer.
func (h *Handler) Init(receiverAddress, interfaceID string) error {
	log.Debugf("Registering logic layer: %s", receiverAddress)
	h.mtx.Lock()
	defer h.mtx.Unlock()

	// already registered?
	s, ok := h.servants[receiverAddress]
	if ok {
		log.Debugf("Logic layer is already registered: %s", receiverAddress)
		// synchronize again with logic layer
		s.command(servantSync{})
		return nil
	}

	// replace receiver addresses
	var addr string
	switch receiverAddress {
	case "xmlrpc_bin://127.0.0.1:31999":
		addr = h.ccuAddr + ":1999"
		log.Debugf("Patched receiver address: %s", addr)
	case "http://127.0.0.1:39292/bidcos":
		addr = h.ccuAddr + ":9292/bidcos"
		log.Debugf("Patched receiver address: %s", addr)
	}

	// create new servant
	s = newServant(addr, interfaceID, h.devices)
	h.servants[receiverAddress] = s

	// synchronize with logic layer
	s.command(servantSync{})
	return nil
}

// Deinit implements DeviceLayer.
func (h *Handler) Deinit(receiverAddress string) error {
	log.Debugf("Unregistering logic layer: %s", receiverAddress)
	h.mtx.Lock()
	defer h.mtx.Unlock()

	// registered?
	s, ok := h.servants[receiverAddress]
	if ok {
		delete(h.servants, receiverAddress)
		h.daemonPool.Run(func(conc.Context) { s.close() })
	} else {
		log.Debugf("Logic layer is NOT registered: %s", receiverAddress)
	}
	return nil
}

// ListDevices implements DeviceLayer.
func (h *Handler) ListDevices() ([]*itf.DeviceDescription, error) {
	devices := h.devices.Devices()
	descr := make([]*itf.DeviceDescription, 0, 50)
	for _, device := range devices {
		descr = append(descr, device.Description())
		channels := device.Channels()
		for _, channel := range channels {
			descr = append(descr, channel.Description())
		}
	}
	return descr, nil
}

// DeleteDevice implements DeviceLayer. Before removing the device from the
// container, deletionNotifier is called.
func (h *Handler) DeleteDevice(address string, flags int) error {
	deviceAddr, channelAddr := itf.SplitAddress(address)
	if channelAddr != "" {
		// ignore deletion of a channel
		log.Debugf("Deletion of channel ignored: %s", address)
		return nil
	}
	h.deletionNotifier(address)
	return h.devices.RemoveDevice(deviceAddr)
}

// GetDeviceDescription implements DeviceLayer.
func (h *Handler) GetDeviceDescription(address string) (*itf.DeviceDescription, error) {
	deviceAddr, channelAddr := itf.SplitAddress(address)
	device, err := h.devices.Device(deviceAddr)
	if err != nil {
		return nil, err
	}
	if channelAddr == "" {
		return device.Description(), nil
	}
	channel, err := device.Channel(channelAddr)
	if err != nil {
		return nil, err
	}
	return channel.Description(), nil
}

// GetParamsetDescription implements DeviceLayer.
func (h *Handler) GetParamsetDescription(address, paramsetKey string) (itf.ParamsetDescription, error) {
	paramset, err := h.getParamset(address, paramsetKey)
	if err != nil {
		return nil, err
	}
	psDescr := make(itf.ParamsetDescription)
	for _, param := range paramset.Parameters() {
		psDescr[param.Description().ID] = param.Description()
	}
	return psDescr, nil
}

// GetParamset implements DeviceLayer.
func (h *Handler) GetParamset(address string, paramsetKey string) (map[string]interface{}, error) {
	paramset, err := h.getParamset(address, paramsetKey)
	if err != nil {
		return nil, err
	}
	values := make(map[string]interface{})
	for _, param := range paramset.Parameters() {
		values[param.Description().ID] = param.Value()
	}
	return values, nil
}

// PutParamset implements DeviceLayer.
func (h *Handler) PutParamset(address string, paramsetKey string, values map[string]interface{}) error {
	paramset, err := h.getParamset(address, paramsetKey)
	if err != nil {
		return err
	}
	for name, value := range values {
		param, err := paramset.Parameter(name)
		if err != nil {
			return err
		}
		err = param.SetValue(value)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetValue implements DeviceLayer.
func (h *Handler) GetValue(address string, valueName string) (interface{}, error) {
	paramset, err := h.getParamset(address, "VALUES")
	if err != nil {
		return nil, err
	}
	param, err := paramset.Parameter(valueName)
	if err != nil {
		return nil, err
	}
	return param.Value(), nil
}

// SetValue implements DeviceLayer.
func (h *Handler) SetValue(address string, valueName string, value interface{}) error {
	paramset, err := h.getParamset(address, "VALUES")
	if err != nil {
		return err
	}
	param, err := paramset.Parameter(valueName)
	if err != nil {
		return err
	}
	return param.SetValue(value)
}

// Ping implements DeviceLayer.
func (h *Handler) Ping(callerID string) (bool, error) {
	h.PublishEvent("CENTRAL", "PONG", callerID)
	return true, nil
}

func (h *Handler) getParamset(address string, paramsetKey string) (GenericParamset, error) {
	deviceAddr, channelAddr := itf.SplitAddress(address)
	device, err := h.devices.Device(deviceAddr)
	if err != nil {
		return nil, err
	}
	if channelAddr == "" {
		switch paramsetKey {
		case "MASTER":
			return device.MasterParamset(), nil
		default:
			return nil, fmt.Errorf("Invalid paramset key for %s: %s", address, paramsetKey)
		}
	}
	channel, err := device.Channel(channelAddr)
	if err != nil {
		return nil, err
	}
	switch paramsetKey {
	case "MASTER":
		return channel.MasterParamset(), nil
	case "VALUES":
		return channel.ValueParamset(), nil
	default:
		return nil, fmt.Errorf("Invalid paramset key for %s: %s", address, paramsetKey)
	}
}
