package vdevices

import (
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/xmlrpc"

	_ "github.com/mdzio/go-lib/testutil"
)

func TestModel(t *testing.T) {
	// *** setup ***

	var onDisposeCalled int32
	var onSetStateCalled int32
	var onDeletionCalled atomic.Value

	// virtual devices container
	vdevs := NewContainer()

	// virtual devices handler
	vdevHandler := NewHandler("", vdevs, func(address string) {
		log.Debugf("OnDelete called: %s", address)
		onDeletionCalled.Store(address)
	})
	defer vdevHandler.Close()
	vdevs.Synchronizer = vdevHandler

	// add a device
	dev := NewDevice("JCK000", "HmIP-MIO16-PCB", vdevHandler)

	// maintenance channel
	NewMaintenanceChannel(dev)

	// switch channel
	sch := NewSwitchChannel(dev)
	sch.OnSetState = func(value bool) bool {
		log.Debugf("Switch channel %s is set: %t", sch.Description().Address, value)
		atomic.AddInt32(&onSetStateCalled, 1)
		return true
	}
	sch.OnDispose = func() {
		log.Debugf("OnDispose called")
		atomic.AddInt32(&onDisposeCalled, 1)
	}

	vdevs.AddDevice(dev)

	// HM RPC dispatcher
	dispatcher := itf.NewDispatcher()
	dispatcher.AddDeviceLayer(vdevHandler)

	// register XML-RPC handler at the HTTP server
	httpHandler := &xmlrpc.Handler{Dispatcher: dispatcher}
	srv := httptest.NewServer(httpHandler)
	defer srv.Close()

	// test client
	cln := itf.DeviceLayerClient{
		Name:   srv.URL,
		Caller: &xmlrpc.Client{Addr: strings.TrimPrefix(srv.URL, "http://")},
	}

	// *** tests ***

	dds, err := cln.ListDevices()
	if err != nil {
		t.Fatal(err)
	}
	if len(dds) != 3 {
		t.Fatal("expected 3 devices")
	}
	if dds[0].Type != "HmIP-MIO16-PCB" || dds[0].Address != "JCK000" ||
		dds[1].Type != "MAINTENANCE" || dds[1].Address != "JCK000:0" ||
		dds[2].Type != "SWITCH" || dds[2].Address != "JCK000:1" {
		t.Fatal("invalid device descriptions")
	}

	dd, err := cln.GetDeviceDescription("JCK000:1")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(dd, dds[2]) {
		t.Fatalf("unexpected device description: %v", dd)
	}

	psd, err := cln.GetParamsetDescription("JCK000:1", "VALUES")
	if err != nil {
		t.Fatal(err)
	} else {
		pd, ok := psd["STATE"]
		if !ok {
			t.Fatal("parameter description STATE missing")
		} else {
			if !reflect.DeepEqual(pd, &itf.ParameterDescription{
				Type: "BOOL", Operations: 7, Flags: 1, Default: false, Max: true, Min: false,
				Control: "SWITCH.STATE", ID: "STATE",
			}) {
				t.Fatal(pd)
			}
		}
	}

	ps, err := cln.GetParamset("JCK000:1", "VALUES")
	if err != nil {
		t.Fatal(err)
	} else {
		if !reflect.DeepEqual(ps, map[string]interface{}{"INSTALL_TEST": false, "STATE": false}) {
			t.Fatal(ps)
		}
	}

	err = cln.PutParamset("JCK000:1", "VALUES", map[string]interface{}{"STATE": true})
	if err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&onSetStateCalled) != 1 {
		t.Fatal("onSetState callback invalid")
	}

	v, err := cln.GetValue("JCK000:1", "STATE")
	if err != nil {
		t.Fatal(err)
	}
	if bv, ok := v.(bool); ok {
		if bv != true {
			t.Fatal(bv)
		}
	} else {
		t.Fatal("expected bool value")
	}

	err = cln.SetValue("JCK000:1", "STATE", false)
	if err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&onSetStateCalled) != 2 {
		t.Fatal("onSetState callback invalid")
	}

	// deletion of a channel should be ignored
	err = cln.DeleteDevice("JCK000:0", 0)
	if err != nil {
		t.Fatal(err)
	}

	err = cln.DeleteDevice("JCK000", 0)
	if err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&onDisposeCalled) != 1 {
		t.Fatal("onDispose callback invalid")
	}
	if onDeletionCalled.Load().(string) != "JCK000" {
		t.Fatal("onDeletion callback invalid")
	}

	dds, err = cln.ListDevices()
	if err != nil {
		t.Fatal(err)
	} else {
		if len(dds) != 0 {
			t.Fatal("expected no devices")
		}
	}
}
