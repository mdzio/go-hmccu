package vdevices

import (
	"sort"

	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"github.com/mdzio/go-lib/conc"
)

const servantQueueSize = 100

type servantSync struct{}

type servantEvent struct {
	address  string
	valueKey string
	value    interface{}
}

type servant struct {
	itfID  string
	client *itf.LogicLayerClient
	model  *Container
	cmds   chan interface{}
	cancel func()
}

func newServant(addr, interfaceID string, model *Container) *servant {
	s := &servant{
		client: &itf.LogicLayerClient{
			Name:   addr,
			Caller: &xmlrpc.Client{Addr: addr},
		},
		itfID: interfaceID,
		model: model,
		cmds:  make(chan interface{}, servantQueueSize),
	}
	s.cancel = conc.DaemonFunc(s.run)
	return s
}

func (s *servant) run(ctx conc.Context) {
	log.Debugf("Starting servant for %s, interface ID %s", s.client.Name, s.itfID)
	for {
		select {
		case cmd := <-s.cmds:
			switch c := cmd.(type) {
			case servantSync:
				// get device list of logic layer
				lds, err := s.client.ListDevices(s.itfID)
				if err != nil {
					log.Errorf("List devices failed on %s, interface ID %s: %v", s.client.Name, s.itfID, err)
					continue
				}
				if ctx.IsDone() {
					return
				}
				// build look up map
				lset := make(map[string]bool)
				for _, ld := range lds {
					lset[ld.Address] = true
				}

				// get device list of device layer
				var dds []*itf.DeviceDescription
				dset := make(map[string]bool)
				for _, dd := range s.model.Devices() {
					dds = append(dds, dd.Description())
					dset[dd.Description().Address] = true
					for _, dch := range dd.Channels() {
						dds = append(dds, dch.Description())
						dset[dch.Description().Address] = true
					}
				}

				// delete devices that no longer exists in the device layer
				var deldev []string
				for _, d := range lds {
					if !dset[d.Address] {
						deldev = append(deldev, d.Address)
					}
				}
				if len(deldev) > 0 {
					// delete channels first
					sort.Sort(sort.Reverse(sort.StringSlice(deldev)))
					s.client.DeleteDevices(s.itfID, deldev)
					if ctx.IsDone() {
						return
					}
				}

				// create devices that are missing in the logic layer
				var newdev []*itf.DeviceDescription
				for _, d := range dds {
					if !lset[d.Address] {
						newdev = append(newdev, d)
					}
				}
				if len(newdev) > 0 {
					s.client.NewDevices(s.itfID, newdev)
				}

			case servantEvent:
				// send event to logic layer
				err := s.client.Event(s.itfID, c.address, c.valueKey, c.value)
				if err != nil {
					log.Errorf("Event failed on %s, interface ID %s: %v", s.client.Name, s.itfID, err)
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

func (s *servant) command(cmd interface{}) {
	select {
	case s.cmds <- cmd:
	default:
		log.Errorf("Queue overflow for %s, interface ID %s", s.client.Name, s.itfID)
	}
}

func (s *servant) close() {
	s.cancel()
}
