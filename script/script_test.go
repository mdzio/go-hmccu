package script

import (
	"testing"

	"github.com/mdzio/go-lib/testutil"
)

const (
	// Test configuration (environment variables)
	// address of the test CCU, e.g. 192.168.0.10
	ccuAddress = "CCU_ADDRESS"

	// CCU system variables (must exist)
	sysVarLogic  = "Sysvar logic"
	sysVarAlarm  = "Sysvar alarm"
	sysVarEnum   = "Sysvar enum"
	sysVarNumber = "Sysvar number"
	sysVarString = "Sysvar string"
)

func TestScriptClient_Execute(t *testing.T) {
	cln := &Client{Addr: testutil.Config(t, ccuAddress)}

	res, err := cln.Execute(`WriteLine("Hello");`)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0] != "Hello" {
		t.Error("unexpected result: ", res)
	}

	svs, err := cln.SystemVariables()
	if err != nil {
		t.Fatal(err)
	}
	var svp *SysVarDef
	for _, sv := range svs {
		if sv.ISEID == "950" {
			svp = sv
			break
		}
	}
	if svp == nil {
		t.Fatal("expected system variable 'presence'")
	}
	if svp.Operations != 7 || svp.Type != "BOOL" {
		t.Error("invalid system variable 'presence'")
	}
}

func TestScriptClient_Rooms(t *testing.T) {
	cln := &Client{Addr: testutil.Config(t, ccuAddress)}

	_, err := cln.Rooms()
	if err != nil {
		t.Fatal(err)
	}
}

func TestScriptClient_Functions(t *testing.T) {
	cln := &Client{Addr: testutil.Config(t, ccuAddress)}

	_, err := cln.Functions()
	if err != nil {
		t.Fatal(err)
	}
}

func TestScriptClient_DevicesAndChannels(t *testing.T) {
	cln := &Client{Addr: testutil.Config(t, ccuAddress)}

	ds, err := cln.Devices()
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) < 2 {
		t.Fatal("expected at least 2 devices")
	}

	cs, err := cln.Channels(ds[0].ISEID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) == 0 {
		t.Fatal("expected at least 1 channel")
	}
}

func TestScriptClient_Programs(t *testing.T) {
	cln := &Client{Addr: testutil.Config(t, ccuAddress)}

	ps, err := cln.Programs()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		t.Logf("%v", p)
	}
}

func TestScriptClient_ReadWriteSysVarTypes(t *testing.T) {
	cln := &Client{Addr: testutil.Config(t, ccuAddress)}
	svs, err := cln.SystemVariables()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		values []interface{}
	}{
		{sysVarLogic, []interface{}{true, false}},
		{sysVarAlarm, []interface{}{true, false}},
		{sysVarEnum, []interface{}{2, 1}},
		{sysVarNumber, []interface{}{42.4, 21.21}},
		{sysVarString, []interface{}{"abc def", "\n", "Line 1\nLine 2"}},
	}
	for _, c := range cases {
		sv := svs.Find(c.name)
		if sv == nil {
			t.Errorf("sysvar %s does not exist", c.name)
			continue
		}
		for _, v := range c.values {
			err := cln.WriteSysVar(sv, v)
			if err != nil {
				t.Error(err)
				continue
			}
			rv, err := cln.ReadSysVars(SysVarDefs{sv})
			if err != nil {
				t.Error(err)
				continue
			}
			if rv[0].Value != v {
				t.Errorf("verify failed for sysvar %s, value %v", c.name, v)
			}
		}
	}
}

func TestScriptClient_ReadMultipleSysVars(t *testing.T) {
	cln := &Client{Addr: testutil.Config(t, ccuAddress)}
	all, err := cln.SystemVariables()
	if err != nil {
		t.Fatal(err)
	}
	// sysvar which should not exist
	svs := SysVarDefs{&SysVarDef{ISEID: "999999"}}
	// sysvars which should work
	for _, n := range []string{sysVarLogic, sysVarAlarm, sysVarEnum, sysVarNumber, sysVarString} {
		sv := all.Find(n)
		if sv == nil {
			t.Fatalf("sysvar %s does not exist", n)
		}
		svs = append(svs, sv)
	}
	res, err := cln.ReadSysVars(svs)
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Err == nil {
		t.Error("expected error")
	}
	for _, r := range res[1:] {
		if r.Err != nil {
			t.Error(r.Err)
		}
	}
}

func TestScriptClient_ReadDeviceValue(t *testing.T) {
	cln := &Client{Addr: testutil.Config(t, ccuAddress)}

	res, err := cln.ReadValues([]ValObjDef{{"BidCos-RF.BidCoS-RF:1.PRESS_SHORT", "ACTION"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res[0].Value.(bool); !ok {
		t.Fatal("invalid type")
	}
	if res[0].Timestamp.IsZero() {
		t.Fatal("invalid timestamp")
	}
}
