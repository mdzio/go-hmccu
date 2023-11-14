package vdevices

import (
	"sort"
	"time"

	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"github.com/mdzio/go-lib/conc"
)

const (
	servantQueueSize  = 200
	servantRetryCount = 6
	servantRetryDelay = 20 * time.Second
)

type servantSync struct{}

type servantEvent struct {
	address  string
	valueKey string
	value    interface{}
}

type servant struct {
	addr, itfID string
	model       *Container
	cmds        chan interface{}
	cancel      func()
}

func newServant(address, interfaceID string, model *Container) *servant {
	s := &servant{
		addr:  address,
		itfID: interfaceID,
		model: model,
		cmds:  make(chan interface{}, servantQueueSize),
	}
	s.cancel = conc.DaemonFunc(s.run)
	return s
}

func (s *servant) run(ctx conc.Context) {
	log.Debugf("Starting servant for %s, interface ID %s", s.addr, s.itfID)
	// use a retrying caller
	cln := &itf.LogicLayerClient{
		Name: s.addr,
		Caller: &xmlrpc.RetryingCaller{
			Caller:     &xmlrpc.Client{Addr: s.addr},
			RetryCount: servantRetryCount,
			RetryDelay: servantRetryDelay,
			Context:    ctx,
		},
	}
	for {
		select {
		case cmd := <-s.cmds:
			switch c := cmd.(type) {
			case servantSync:
				// get device list of logic layer
				lds, err := cln.ListDevices(s.itfID)
				if err != nil {
					log.Errorf("List devices failed on %s, interface ID %s: %v", s.addr, s.itfID, err)
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
					cln.DeleteDevices(s.itfID, deldev)
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
					cln.NewDevices(s.itfID, newdev)
				}

			case servantEvent:
				// send event to logic layer
				err := cln.Event(s.itfID, c.address, c.valueKey, c.value)
				if err != nil {
					log.Errorf("Event failed on %s, interface ID %s: %v", s.addr, s.itfID, err)
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
		log.Errorf("Queue overflow for %s, interface ID %s", s.addr, s.itfID)
	}
}

func (s *servant) close() {
	s.cancel()
}
