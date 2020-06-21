package script

import (
	"os"
	"strings"
	"testing"

	"github.com/mdzio/go-logging"
)

func init() {
	var l logging.LogLevel
	err := l.Set(os.Getenv("LOG_LEVEL"))
	if err == nil {
		logging.SetLevel(l)
	}
}

func ccuAddr(t *testing.T) string {
	addr := os.Getenv("CCU_ADDRESS")
	if len(addr) == 0 {
		t.Skip("env variable CCU_ADDRESS not set")
	}
	return addr
}

// button with rooms roomBathroom, roomBedroom and functions funcButton, funcCentral.
func buttonChAddr(t *testing.T) string {
	addr := os.Getenv("BUTTON_CHANNEL_ADDRESS")
	if len(addr) == 0 {
		t.Skip("env variable BUTTON_CHANNEL_ADDRESS not set")
	}
	return addr
}

func buttonFullAddr(t *testing.T) string {
	addr := os.Getenv("BUTTON_FULL_ADDRESS")
	if len(addr) == 0 {
		t.Skip("env variable BUTTON_FULL_ADDRESS not set")
	}
	return addr
}

func TestScriptClient_Execute(t *testing.T) {
	cln := &Client{Addr: ccuAddr(t)}

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
	if svp.ISEID == "0" {
		t.Error("expected system variable 'presence'")
	}
	if svp.Operations != 7 || svp.Type != "BOOL" {
		t.Error("invalid system variable 'presence'")
	}
}

func TestScriptClient_Rooms(t *testing.T) {
	cln := &Client{Addr: ccuAddr(t)}

	_, err := cln.Rooms()
	if err != nil {
		t.Fatal(err)
	}
}

func TestScriptClient_Functions(t *testing.T) {
	cln := &Client{Addr: ccuAddr(t)}

	_, err := cln.Functions()
	if err != nil {
		t.Fatal(err)
	}
}

func TestScriptClient_DevicesAndChannels(t *testing.T) {
	cln := &Client{Addr: ccuAddr(t)}

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

func TestScriptClient_SystemVariables(t *testing.T) {
	cln := &Client{Addr: ccuAddr(t)}

	svs, err := cln.SystemVariables()
	if err != nil {
		t.Fatal(err)
	}
	for _, sv := range svs {
		if strings.HasPrefix(sv.Name, "Systemvariable ") {
			v, _, _, err := cln.ReadSysVar(sv)
			if err != nil {
				t.Fatal(err)
			}

			switch sv.Type {
			case "BOOL":
				fallthrough
			case "ALARM":
				b := v.(bool)
				err = cln.WriteSysVar(sv, !b)
				if err != nil {
					t.Fatal(err)
				}
			case "ENUM":
				i := v.(int)
				err = cln.WriteSysVar(sv, i+1)
				if err != nil {
					t.Fatal(err)
				}
			case "FLOAT":
				f := v.(float64)
				err = cln.WriteSysVar(sv, f+1.234)
				if err != nil {
					t.Fatal(err)
				}
			case "STRING":
				s := v.(string)
				err = cln.WriteSysVar(sv, s+"Test")
				if err != nil {
					t.Fatal(err)
				}
			}

			v, _, _, err = cln.ReadSysVar(sv)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestScriptClient_Value(t *testing.T) {
	cln := &Client{Addr: ccuAddr(t)}

	v, ts, _, err := cln.ReadValue("\""+buttonFullAddr(t)+"\"", "ACTION")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v.(bool); !ok {
		t.Fatal("invalid type")
	}
	if ts.IsZero() {
		t.Fatal("invalid timestamp")
	}
}

func TestScriptClient_EnumPrograms(t *testing.T) {
	cln := &Client{Addr: ccuAddr(t)}

	ps, err := cln.Programs()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		t.Logf("%v", p)
	}
}
