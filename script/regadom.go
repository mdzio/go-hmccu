package script

import (
	"sync/atomic"
	"time"
)

const (
	// exploration cycle for the ReGa DOM
	reGaDomExploreCycle = 30 * time.Minute

	// delay between ReGaHss requests while exploring
	reGaHssDelay = 50 * time.Millisecond
)

type model struct {
	rooms     map[string]AspectDef  // key: ISEID
	functions map[string]AspectDef  // key: ISEID
	devices   map[string]DeviceDef  // key: Address
	channels  map[string]ChannelDef // key: Address
}

// ReGaDOM retrieves and caches information (e.g. rooms, functions) from the ReGa DOM of the CCU.
type ReGaDOM struct {
	ScriptClient *Client

	model atomic.Value

	timer       *time.Timer
	stopRequest chan struct{}
	stopped     chan struct{}
	refresh     chan struct{}
}

// NewReGaDOM creates a new ReGaDOM.
func NewReGaDOM(scriptClient *Client) *ReGaDOM {
	r := &ReGaDOM{
		ScriptClient: scriptClient,
		stopRequest:  make(chan struct{}),
		stopped:      make(chan struct{}),
		refresh:      make(chan struct{}, 1),
	}
	r.model.Store(model{})
	return r
}

// Start starts the exploration of the ReGa DOM.
func (rd *ReGaDOM) Start() {
	// start ReGa DOM explorer
	go func() {
		scriptLog.Info("Starting ReGa DOM explorer")

		// defer clean up
		defer func() {
			scriptLog.Debug("Stopping ReGa DOM explorer")
			rd.stopped <- struct{}{}
		}()

		// exploration cycle
		for {
			if rd.explore() {
				return
			}
			rd.timer = time.NewTimer(reGaDomExploreCycle)
			select {
			case <-rd.stopRequest:
				// clean up timer
				if !rd.timer.Stop() {
					<-rd.timer.C
				}
				return
			case <-rd.timer.C:
				// loop
			case <-rd.refresh:
				// loop
			}
		}
	}()
}

// Stop stops the exploration of the ReGa DOM.
func (rd *ReGaDOM) Stop() {
	// stop exploration of ReGa DOM
	rd.stopRequest <- struct{}{}
	<-rd.stopped
}

// Refresh triggers a reexploration of the ReGa DOM.
func (rd *ReGaDOM) Refresh() {
	select {
	case rd.refresh <- struct{}{}:
	default:
	}
}

func (rd *ReGaDOM) delay() bool {
	t := time.NewTimer(reGaHssDelay)
	select {
	case <-rd.stopRequest:
		// clean up timer
		if !t.Stop() {
			<-t.C
		}
		return true
	case <-t.C:
		return false
	}
}

// returns true, if the exploration cycle should be stopped
func (rd *ReGaDOM) explore() bool {
	scriptLog.Debug("Exploring ReGa DOM")

	// build new model
	model := model{}
	model.rooms = make(map[string]AspectDef)
	model.functions = make(map[string]AspectDef)
	model.devices = make(map[string]DeviceDef)
	model.channels = make(map[string]ChannelDef)

	// retrieve rooms
	rs, err := rd.ScriptClient.Rooms()
	if err != nil {
		scriptLog.Error("Retrieving of rooms from the CCU failed: ", err)
		return false
	}
	if rd.delay() {
		return true
	}
	for _, r := range rs {
		model.rooms[r.ISEID] = r
	}

	// retrieve functions
	fs, err := rd.ScriptClient.Functions()
	if err != nil {
		scriptLog.Error("Retrieving of functions from the CCU failed: ", err)
		return false
	}
	if rd.delay() {
		return true
	}
	for _, f := range fs {
		model.functions[f.ISEID] = f
	}

	// retrieve devices
	ds, err := rd.ScriptClient.Devices()
	if err != nil {
		scriptLog.Error("Retrieving of devices from the CCU failed: ", err)
		return false
	}
	if rd.delay() {
		return true
	}
	for _, d := range ds {
		model.devices[d.Address] = d

		// retrieve channels
		cs, err := rd.ScriptClient.Channels(d.ISEID)
		if err != nil {
			scriptLog.Error("Retrieving of devices from the CCU failed: ", err)
			return false
		}
		if rd.delay() {
			return true
		}
		for _, c := range cs {
			// store channel
			model.channels[c.Address] = c
			// add to rooms
			for _, rid := range c.Rooms {
				if r, ok := model.rooms[rid]; ok {
					r.Channels = append(r.Channels, c.Address)
					model.rooms[rid] = r
				}
			}
			// add to function
			for _, fid := range c.Functions {
				if f, ok := model.functions[fid]; ok {
					f.Channels = append(f.Channels, c.Address)
					model.functions[fid] = f
				}
			}
		}
	}

	// activate model
	rd.model.Store(model)
	scriptLog.Debug("Exploring ReGa DOM completed")
	return false
}

// Room returns info about a room.
func (rd *ReGaDOM) Room(iseID string) *AspectDef {
	tm := rd.model.Load()
	model := tm.(model)
	r, ok := model.rooms[iseID]
	if !ok {
		return nil
	}
	return &r
}

// Rooms returns info about all rooms.
func (rd *ReGaDOM) Rooms() map[string]AspectDef {
	tm := rd.model.Load()
	model := tm.(model)
	return model.rooms
}

// Function returns info about a function.
func (rd *ReGaDOM) Function(iseID string) *AspectDef {
	tm := rd.model.Load()
	model := tm.(model)
	f, ok := model.functions[iseID]
	if !ok {
		return nil
	}
	return &f
}

// Functions returns info about all functions.
func (rd *ReGaDOM) Functions() map[string]AspectDef {
	tm := rd.model.Load()
	model := tm.(model)
	return model.functions
}

// Device returns info about a device.
func (rd *ReGaDOM) Device(addr string) *DeviceDef {
	tm := rd.model.Load()
	model := tm.(model)
	d, ok := model.devices[addr]
	if !ok {
		return nil
	}
	return &d
}

// Channel returns info about a channel.
func (rd *ReGaDOM) Channel(addr string) *ChannelDef {
	tm := rd.model.Load()
	model := tm.(model)
	c, ok := model.channels[addr]
	if !ok {
		return nil
	}
	return &c
}
