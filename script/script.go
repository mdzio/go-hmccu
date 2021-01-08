package script

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/mdzio/go-logging"

	"golang.org/x/text/encoding/charmap"
)

const (
	// max. size of a valid response, if not specified: 10 MB
	// (max. size of a single response line is always 64 KB)
	scriptRespLimit = 10 * 1024 * 1024
)

const enumAspectsScript = `! Enumerating aspects
object eobj = dom.GetObject({{ . }});
if (eobj) {
	WriteLine("OK");
	string id;
	foreach (id, eobj.EnumIDs()) {
		object obj = dom.GetObject(id);
		WriteLine(obj.ID() # "\t" # obj.Name() # "\t" # obj.EnumInfo());
	}
} else {
	WriteLine("Object not found or has wrong type");
}`

const enumDevicesScript = `! Enumerating devices
object eobj = dom.GetObject(ID_DEVICES);
if (eobj) {
	WriteLine("OK");
	string id;
	foreach (id, eobj.EnumIDs()) {
		object obj = dom.GetObject(id);
		WriteLine(obj.ID() # "\t" # obj.Name() # "\t" # obj.Address());
	}
} else {
	WriteLine("Object not found");
}`

const enumChannelsScript = `! Enumerating channels
object dobj = dom.GetObject({{ . }});
if (dobj && dobj.Type()==OT_DEVICE) {
	WriteLine("OK");
	string cid; foreach(cid, dobj.Channels()) {
		var cobj=dom.GetObject(cid);
		WriteLine(cobj.ID() # "\t" # cobj.Name() # "\t" # cobj.Address());
		WriteLine(cobj.ChnRoom());
		WriteLine(cobj.ChnFunction());
	}
} else {
	WriteLine("Object not found or has wrong type");
}`

const enumProgramsScript = `! Enumerating programs
object eobj = dom.GetObject(ID_PROGRAMS);
if (eobj) {
	WriteLine("OK");
	string id;
	foreach (id, eobj.EnumIDs()) {
		object obj = dom.GetObject(id);
		WriteLine(obj.ID() # "\t" # obj.Name() # "\t" # obj.PrgInfo() # "\t" # obj.Active() # "\t" # obj.Visible());
	}
} else {
	WriteLine("Object not found");
}`

const execProgramScript = `! Executing program
object pobj = dom.GetObject({{ . }});
if (pobj && pobj.Type()==OT_PROGRAM) {
	pobj.ProgramExecute();
	WriteLine("OK");
} else {
	WriteLine("Object not found or has wrong type");
}`

const readExecTimeScript = `! Reading last execution time of program
object pobj = dom.GetObject({{ . }});
if (pobj && pobj.Type()==OT_PROGRAM) {
	WriteLine("OK");
	WriteLine(pobj.ProgramLastExecuteTime());	
} else {
	WriteLine("Object not found or has wrong type");
}`

const enumSysVarsScript = `! Enumerating system variables
string id; foreach(id, dom.GetObject(ID_SYSTEM_VARIABLES).EnumIDs()) {
	var sv=dom.GetObject(id);
	var vt=sv.ValueType(); var st=sv.ValueSubType();
	var outvt="";
	if ((vt==ivtBinary) && (st==istBool)) { outvt="BOOL"; }
	if ((vt==ivtBinary) && (st==istAlarm)) { outvt="ALARM"; }
	if ((vt==ivtInteger) && (st==istEnum)) { outvt="ENUM"; }
	if ((vt==ivtFloat) && (st==istGeneric)) { outvt="FLOAT"; }
	if ((vt==ivtString) && (st==istChar8859)) { outvt="STRING"; }
	var dpinfo=sv.DPInfo().Replace("\t", " ").Replace("\r\n", " ").Replace("\r", " ").Replace("\n", " ");
	if (outvt!="") { WriteLine(id # "\t" # sv.Name() # "\t" # dpinfo # "\t" # sv.ValueMax() # "\t" #
		sv.ValueUnit() # "\t" # sv.ValueMin() # "\t" # sv.Operations() # "\t" # outvt # "\t" #
		sv.ValueName0() # "\t" # sv.ValueName1() # "\t" # sv.ValueList()); }
}`

const readValueScript = `! Reading value
var sv=dom.GetObject({{ . }});
if (sv) {
	if (sv.IsTypeOf(OT_DP) || sv.IsTypeOf(OT_VARDP) || sv.IsTypeOf(OT_ALARMDP)) {
		WriteLine("OK"); 
		WriteLine(sv.Timestamp().ToInteger());
		WriteLine(sv.Value()); 
	} else {
		WriteLine("Object has wrong type");
	}
} else {
	WriteLine("Not found");
}`

// readValuesScript expects as dot parameter a tab separated string of object
// IDs. Special characters in string data points are returned percent encoded.
const readValuesScript = `! Reading multiple values
string id; foreach(id,"{{ . }}") {
	var dp=dom.GetObject(id);
	if (dp) {
	  if (dp.IsTypeOf(OT_DP) || dp.IsTypeOf(OT_VARDP) || dp.IsTypeOf(OT_ALARMDP)) {
		WriteLine("OK"); 
		WriteLine(dp.Timestamp().ToInteger());
		WriteLine(dp.Value().ToString().Replace("%", "%25").Replace("\n", "%0A"));
	  } else {
		WriteLine("Object has wrong type");
	  }
	} else {
	  WriteLine("Not found");
	}
}`

const writeValueScript = `! Writing value
var sv=dom.GetObject({{ .ISEID }});
if (sv) {
	if (sv.IsTypeOf(OT_DP) || sv.IsTypeOf(OT_VARDP) || sv.IsTypeOf(OT_ALARMDP)) {
		sv.State({{ .Value }});
		WriteLine("OK"); 
	} else {
		WriteLine("Object has wrong type");
	}
} else {
	WriteLine("Not found");
}`

var (
	scriptLog = logging.Get("script-client")

	enumAspectsTempl  = template.Must(template.New("enumAspects").Parse(enumAspectsScript))
	enumDevicesTempl  = template.Must(template.New("enumDevices").Parse(enumDevicesScript))
	enumChannelsTempl = template.Must(template.New("enumChannels").Parse(enumChannelsScript))
	enumProgramsTempl = template.Must(template.New("enumPrograms").Parse(enumProgramsScript))
	execProgramTempl  = template.Must(template.New("execProgram").Parse(execProgramScript))
	readExecTimeTempl = template.Must(template.New("readExecTime").Parse(readExecTimeScript))
	enumSysVarsTempl  = template.Must(template.New("enumSysVars").Parse(enumSysVarsScript))
	readValueTempl    = template.Must(template.New("readValue").Parse(readValueScript))
	readValuesTempl   = template.Must(template.New("readValues").Parse(readValuesScript))
	writeValueTempl   = template.Must(template.New("writeValue").Parse(writeValueScript))
)

// SysVarDef contains meta data about a ReGaHss system variable.
type SysVarDef struct {
	ISEID       string
	Name        string
	Description string
	Unit        string
	Operations  int
	Type        string

	// type: FLOAT
	Minimum *float64
	Maximum *float64

	// type: ALARM or BOOL
	ValueName0 *string
	ValueName1 *string

	// type: ENUM
	ValueList *[]string
}

// String implements fmt.Stringer.
func (sv *SysVarDef) String() string {
	var b strings.Builder
	b.WriteString("reGaHssID: ")
	b.WriteString(sv.ISEID)
	b.WriteString(", name: ")
	b.WriteString(sv.Name)
	b.WriteString(", description: ")
	b.WriteString(sv.Description)
	b.WriteString(", unit: ")
	b.WriteString(sv.Unit)
	b.WriteString(", operations: ")
	b.WriteString(strconv.Itoa(sv.Operations))
	b.WriteString(", type: ")
	b.WriteString(sv.Type)
	if sv.Minimum != nil {
		b.WriteString(", minimum: ")
		b.WriteString(strconv.FormatFloat(*sv.Minimum, 'G', -1, 64))
	}
	if sv.Maximum != nil {
		b.WriteString(", maximum: ")
		b.WriteString(strconv.FormatFloat(*sv.Maximum, 'G', -1, 64))
	}
	if sv.ValueName0 != nil {
		b.WriteString(", valueName0: ")
		b.WriteString(*sv.ValueName0)
	}
	if sv.ValueName1 != nil {
		b.WriteString(", valueName1: ")
		b.WriteString(*sv.ValueName1)
	}
	if sv.ValueList != nil {
		b.WriteString(", valueList: ")
		b.WriteString(strings.Join(*sv.ValueList, ";"))
	}
	return b.String()
}

// Equal compares this system variable definition with another one.
func (sv *SysVarDef) Equal(o *SysVarDef) bool {
	if sv.ISEID != o.ISEID {
		return false
	}
	if sv.Name != o.Name {
		return false
	}
	if sv.Description != o.Description {
		return false
	}
	if sv.Unit != o.Unit {
		return false
	}
	if sv.Operations != o.Operations {
		return false
	}
	if sv.Type != o.Type {
		return false
	}
	if e := optFloat64Equal(sv.Minimum, o.Minimum); !e {
		return false
	}
	if e := optFloat64Equal(sv.Maximum, o.Maximum); !e {
		return false
	}
	if e := optStringEqual(sv.ValueName0, o.ValueName0); !e {
		return false
	}
	if e := optStringEqual(sv.ValueName1, o.ValueName1); !e {
		return false
	}
	if (sv.ValueList == nil) != (o.ValueList == nil) {
		return false
	}
	if sv.ValueList != nil {
		if len(*sv.ValueList) != len(*o.ValueList) {
			return false
		}
		for i := range *sv.ValueList {
			if (*sv.ValueList)[i] != (*o.ValueList)[i] {
				return false
			}
		}
	}
	return true
}

// SysVarDefs is a slice of SysVarDef.
type SysVarDefs []*SysVarDef

// Find finds a system variable by name. If not found, nil is returned.
// SysVarDefs must be sorted.
func (s SysVarDefs) Find(name string) *SysVarDef {
	i := sort.Search(len(s), func(i int) bool { return s[i].Name >= name })
	if i < len(s) && s[i].Name == name {
		return s[i]
	}
	return nil
}

// AspectDef describes a room or function of a channel.
type AspectDef struct {
	ISEID       string
	DisplayName string
	Comment     string
	// Channels will not be returned by Rooms() or Functions()!
	// ReGaDOM.explore() sets this member with a reverse lookup.
	Channels []string // channel address
}

func responseToAspects(resp []string) ([]AspectDef, error) {
	if len(resp) < 1 {
		return nil, errors.New("Retrieving rooms/channels: Expected at least one response line")
	}
	if resp[0] != "OK" {
		return nil, fmt.Errorf("Retrieving rooms/channels: HM script signals error: %s", resp[0])
	}
	var as []AspectDef
	for _, l := range resp[1:] {
		fs := strings.Split(l, "\t")
		if len(fs) != 3 {
			return nil, fmt.Errorf("Retrieving rooms/channels: Invalid response line: %s", l)
		}
		as = append(as, AspectDef{ISEID: fs[0], DisplayName: fs[1], Comment: fs[2]})
	}
	return as, nil
}

// DeviceDef describes a device.
type DeviceDef struct {
	ISEID       string
	DisplayName string
	Address     string
}

// ChannelDef describes a channel.
type ChannelDef struct {
	ISEID       string
	DisplayName string
	Address     string
	Rooms       []string // ISEID's
	Functions   []string // ISEID's
}

// ProgramDef describes a program in the ReGaHss.
type ProgramDef struct {
	ISEID       string
	DisplayName string
	Description string
	Active      bool
	Visible     bool
}

// Client executes HM scripts remotely on the CCU.
type Client struct {
	// IP address or network name of the CCU
	Addr string

	// Limits the size of a valid response
	RespLimit int64
}

// Execute remotely executes a HM script on the CCU.
func (sc *Client) Execute(script string) ([]string, error) {
	scriptLog.Trace("Executing HM script: ", script)

	// encode request body with ISO8859-1
	var reqBuf bytes.Buffer
	reqWriter := charmap.ISO8859_1.NewEncoder().Writer(&reqBuf)
	reqWriter.Write([]byte(script))

	// http post
	addr := "http://" + sc.Addr + ":8181/tclrega.exe"
	httpResp, err := http.Post(addr, "", bytes.NewReader(reqBuf.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed on %s: %v", addr, err)
	}
	defer httpResp.Body.Close()

	// check status
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 299 {
		return nil, fmt.Errorf("HTTP request failed on %s with code: %s", addr, httpResp.Status)
	}

	// limit response size
	limit := sc.RespLimit
	if limit == 0 {
		limit = scriptRespLimit
	}
	limitReader := io.LimitReader(httpResp.Body, limit)

	// decode response body with ISO8859-1
	decReader := charmap.ISO8859_1.NewDecoder().Reader(limitReader)

	// read response and split lines
	scn := bufio.NewScanner(decReader)
	var resp []string
	for scn.Scan() {
		l := scn.Text()
		if !strings.HasPrefix(l, "<xml><exec>") {
			resp = append(resp, l)
		}
	}
	if scn.Err() != nil {
		return nil, fmt.Errorf("Parsing of response failed from %s: %v", addr, scn.Err())
	}
	if scriptLog.TraceEnabled() {
		scriptLog.Trace("HM script response: ", strings.Join(resp, "\\n"))
	}
	return resp, nil
}

// ExecuteTempl executes a HM script template with the specified data remotely on the CCU.
func (sc *Client) ExecuteTempl(templ *template.Template, data interface{}) ([]string, error) {
	// fill template
	var sb strings.Builder
	err := templ.Execute(&sb, data)
	if err != nil {
		return nil, fmt.Errorf("Rendering of HM template with data %v failed: %v", data, err)
	}

	// execute script
	resp, err := sc.Execute(sb.String())
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Rooms retrieves the room list from the CCU.
func (sc *Client) Rooms() ([]AspectDef, error) {
	scriptLog.Debug("Retrieving rooms")
	resp, err := sc.ExecuteTempl(enumAspectsTempl, "ID_ROOMS")
	if err != nil {
		return nil, err
	}
	return responseToAspects(resp)
}

// Functions retrieves the room list from the CCU.
func (sc *Client) Functions() ([]AspectDef, error) {
	scriptLog.Debug("Retrieving functions")
	resp, err := sc.ExecuteTempl(enumAspectsTempl, "ID_FUNCTIONS")
	if err != nil {
		return nil, err
	}
	return responseToAspects(resp)
}

// Devices retrieves all devices from the CCU.
func (sc *Client) Devices() ([]DeviceDef, error) {
	scriptLog.Debug("Retrieving devices")
	resp, err := sc.ExecuteTempl(enumDevicesTempl, nil)
	if err != nil {
		return nil, err
	}
	if len(resp) < 1 {
		return nil, errors.New("Retrieving devices: Expected at least one response line")
	}
	if resp[0] != "OK" {
		return nil, fmt.Errorf("Retrieving devices: HM script signals error: %s", resp[0])
	}
	var ds []DeviceDef
	for _, l := range resp[1:] {
		fs := strings.Split(l, "\t")
		if len(fs) != 3 {
			return nil, fmt.Errorf("Retrieving devices: Invalid response line: %s", l)
		}
		ds = append(ds, DeviceDef{ISEID: fs[0], DisplayName: fs[1], Address: fs[2]})
	}
	return ds, nil
}

// Channels retrieves the channels of a device from the CCU.
func (sc *Client) Channels(iseID string) ([]ChannelDef, error) {
	scriptLog.Debugf("Retrieving channels of device: %s", iseID)
	resp, err := sc.ExecuteTempl(enumChannelsTempl, iseID)
	if err != nil {
		return nil, err
	}
	if len(resp) < 1 {
		return nil, fmt.Errorf("Retrieving channels of device %s: Expected at least one response line", iseID)
	}
	if resp[0] != "OK" {
		return nil, fmt.Errorf("Retrieving channels of device %s: HM script signals error: %s", iseID, resp[0])
	}
	var cs []ChannelDef
	for l := 1; l < len(resp); l += 3 {
		if l+2 >= len(resp) {
			return nil, fmt.Errorf("Retrieving channels of device %s: Remaining lines are not complete", iseID)
		}
		fields := strings.Split(resp[l], "\t")
		rooms := strings.Split(resp[l+1], "\t")
		if rooms[0] == "" {
			rooms = nil
		}
		funcs := strings.Split(resp[l+2], "\t")
		if funcs[0] == "" {
			funcs = nil
		}
		cs = append(cs,
			ChannelDef{
				ISEID:       fields[0],
				DisplayName: fields[1],
				Address:     fields[2],
				Rooms:       rooms,
				Functions:   funcs,
			},
		)
	}
	return cs, nil
}

// SystemVariables retrieves the list of system variables in the ReGaHss.
// SysVarDefs is returned sorted.
func (sc *Client) SystemVariables() (SysVarDefs, error) {
	scriptLog.Debug("Retrieving list of system variables")

	// query ReGaHss
	lines, err := sc.ExecuteTempl(enumSysVarsTempl, nil)
	if err != nil {
		return nil, fmt.Errorf("Retrieving list of system variables failed: %v", err)
	}

	// parse response
	var sysvars SysVarDefs
	for _, l := range lines {
		fs := strings.Split(l, "\t")
		if len(fs) == 11 {
			var sv SysVarDef
			// ReGaHss id
			sv.ISEID = fs[0]
			// name
			sv.Name = fs[1]
			// description
			sv.Description = fs[2]
			// unit
			sv.Unit = fs[4]
			// operations
			op, err := strconv.Atoi(fs[6])
			if err != nil {
				scriptLog.Warning("Retrieving list of system variables: Invalid operations: ", l)
				continue
			}
			sv.Operations = op
			// type
			sv.Type = fs[7]
			// fields for specific data types
			switch sv.Type {
			case "FLOAT":
				min, err := strconv.ParseFloat(fs[5], 64)
				if err != nil {
					scriptLog.Warning("Retrieving list of system variables: Invalid minimum: ", l)
					continue
				}
				sv.Minimum = &min
				max, err := strconv.ParseFloat(fs[3], 64)
				if err != nil {
					scriptLog.Warning("Retrieving list of system variables: Invalid maximum: ", l)
					continue
				}
				sv.Maximum = &max
			case "ALARM":
				fallthrough
			case "BOOL":
				sv.ValueName0 = &fs[8]
				sv.ValueName1 = &fs[9]
			case "ENUM":
				l := strings.Split(fs[10], ";")
				sv.ValueList = &l
			}
			sysvars = append(sysvars, &sv)
		} else {
			scriptLog.Warning("Retrieving list of system variables: Invalid response line: ", l)
		}
	}

	// sort by name for quick lookup
	sort.Slice(sysvars, func(i, j int) bool { return sysvars[i].Name < sysvars[j].Name })

	return sysvars, nil
}

// ValObjDef identifies a ReGaDom value object and its data type.
type ValObjDef struct {
	ISEID, Type string
}

// Value is the result of reading the value of an ReGaDom object.
type Value struct {
	Value     interface{}
	Timestamp time.Time
	Uncertain bool
	Err       error
}

// ReadValues reads values of multiple ReGaDOM objects.
func (sc *Client) ReadValues(objs []ValObjDef) ([]Value, error) {
	// build tab separated list of IDs
	sb := strings.Builder{}
	first := true
	for _, obj := range objs {
		if first {
			first = false
		} else {
			sb.WriteRune('\t')
		}
		sb.WriteString(obj.ISEID)
	}
	ids := sb.String()
	if scriptLog.DebugEnabled() {
		scriptLog.Debug("Reading values of objects: ", strings.ReplaceAll(ids, "\t", " "))
	}

	// execute script
	resp, err := sc.ExecuteTempl(readValuesTempl, ids)
	if err != nil {
		return nil, fmt.Errorf("Reading object values failed: %v", err)
	}

	// parse result
	result := make([]Value, len(objs))
	line := 0
	for idx := range objs {
		// unexpected end of response?
		if line >= len(resp) || (resp[line] == "OK" && line+2 >= len(resp)) {
			return nil, errors.New("Reading object values failed: Unexpected end of response")
		}

		// HM script error?
		if resp[line] != "OK" {
			result[idx].Err = errors.New(resp[line])
			line++
			continue
		}

		// parse timestamp
		sec, err := strconv.ParseInt(resp[line+1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Reading value of %s failed: Invalid timestamp: %s", objs[idx].ISEID, resp[line+1])
		}
		ts := time.Unix(sec, 0)
		result[idx].Timestamp = ts
		// uncertain?
		if sec == 0 {
			result[idx].Uncertain = true
		}

		// parse value
		strval, err := url.PathUnescape(resp[line+2])
		if err != nil {
			return nil, fmt.Errorf("Reading value of %s failed: Invalid percent encoding: %s", objs[idx].ISEID, strval)
		}
		switch objs[idx].Type {
		case "BOOL":
			fallthrough
		case "ALARM":
			fallthrough
		case "ACTION":
			if strval == "" {
				result[idx].Value = false
				result[idx].Uncertain = true
			} else {
				value, err := strconv.ParseBool(strval)
				if err != nil {
					return nil, fmt.Errorf("Reading value of %s failed: Invalid BOOL/ALARM/ACTION value: %s", objs[idx].ISEID, strval)
				}
				result[idx].Value = value
			}

		case "INTEGER":
			fallthrough
		case "ENUM":
			if strval == "" {
				result[idx].Value = 0
				result[idx].Uncertain = true
			} else {
				tmp, err := strconv.ParseInt(strval, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("Reading value of %s failed: Invalid INTEGER/ENUM value: %s", objs[idx].ISEID, strval)
				}
				result[idx].Value = int(tmp)
			}

		case "FLOAT":
			if strval == "" {
				result[idx].Value = 0.0
				result[idx].Uncertain = true
			} else {
				value, err := strconv.ParseFloat(strval, 64)
				if err != nil {
					return nil, fmt.Errorf("Reading value of %s failed: Invalid FLOAT value: %s", objs[idx].ISEID, strval)
				}
				result[idx].Value = value
			}

		case "STRING":
			result[idx].Value = strval

		default:
			return nil, fmt.Errorf("Reading value of %s failed: Unsupported type: %s", objs[idx].ISEID, objs[idx].Type)
		}
		line += 3
	}
	return result, nil
}

// WriteValue sets the value of a ReGaDOM object.
func (sc *Client) WriteValue(obj ValObjDef, value interface{}) error {
	scriptLog.Debugf("Writing value %v to object %s", value, obj.ISEID)

	// convert value
	var strval string
	switch obj.Type {
	case "BOOL":
		fallthrough
	case "ALARM":
		fallthrough
	case "ACTION":
		b, ok := value.(bool)
		if !ok {
			return fmt.Errorf("Writing of object %s failed: Invalid type for BOOL/ALARM/ACTION: %#v", obj.ISEID, value)
		}
		strval = fmt.Sprint(b)

	case "INTEGER":
		fallthrough
	case "ENUM":
		i, ok := value.(int)
		if !ok {
			return fmt.Errorf("Writing of object %s failed: Invalid type for INTEGER/ENUM: %#v", obj.ISEID, value)
		}
		strval = fmt.Sprint(i)

	case "FLOAT":
		f, ok := value.(float64)
		if !ok {
			return fmt.Errorf("Writing of object %s failed: Invalid type for FLOAT: %#v", obj.ISEID, value)
		}
		// 6 decimal places are supported
		strval = fmt.Sprintf("%f", f)

	case "STRING":
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("Writing of object %s failed: Invalid type for STRING: %#v", obj.ISEID, value)
		}
		strval = strconv.Quote(s)

	default:
		return fmt.Errorf("Writing of object %s failed: Unsupported type: %s", obj.ISEID, obj.Type)
	}

	// execute script
	resp, err := sc.ExecuteTempl(writeValueTempl, map[string]interface{}{"ISEID": obj.ISEID, "Value": strval})
	if err != nil {
		return fmt.Errorf("Writing of object %s failed: %v", obj.ISEID, err)
	}
	if len(resp) != 1 {
		return fmt.Errorf("Writing of object %s failed: Expected one response line", obj.ISEID)
	}
	if resp[0] != "OK" {
		return fmt.Errorf("Writing of object %s failed: HM script signals error: %s", obj.ISEID, resp[0])
	}
	return nil
}

// ReadSysVars reads the values of system variables.
func (sc *Client) ReadSysVars(sysVars SysVarDefs) ([]Value, error) {
	valObjs := make([]ValObjDef, len(sysVars))
	for idx, sysVar := range sysVars {
		valObjs[idx] = ValObjDef{sysVar.ISEID, sysVar.Type}
	}
	return sc.ReadValues(valObjs)
}

// WriteSysVar sets the value of a system variable.
func (sc *Client) WriteSysVar(sysVar *SysVarDef, value interface{}) error {
	return sc.WriteValue(ValObjDef{sysVar.ISEID, sysVar.Type}, value)
}

// Programs retrieves all programs from the CCU.
func (sc *Client) Programs() ([]*ProgramDef, error) {
	scriptLog.Debug("Retrieving programs")
	resp, err := sc.ExecuteTempl(enumProgramsTempl, nil)
	if err != nil {
		return nil, err
	}
	if len(resp) < 1 {
		return nil, errors.New("Retrieving programs: Expected at least one response line")
	}
	if resp[0] != "OK" {
		return nil, fmt.Errorf("Retrieving programs: HM script signals error: %s", resp[0])
	}
	var ps []*ProgramDef
	for _, l := range resp[1:] {
		fs := strings.Split(l, "\t")
		if len(fs) != 5 {
			return nil, fmt.Errorf("Retrieving programs: Invalid response line: %s", l)
		}
		// fields: ID, Name, PrgInfo, Active, Visible
		ps = append(ps, &ProgramDef{
			ISEID:       fs[0],
			DisplayName: fs[1],
			Description: fs[2],
			Active:      fs[3] == "true",
			Visible:     fs[4] == "true",
		})
	}
	return ps, nil
}

// ExecProgram executes a ReGaHssProgram.
func (sc *Client) ExecProgram(p *ProgramDef) error {
	scriptLog.Debug("Executing program: ", p.DisplayName)
	resp, err := sc.ExecuteTempl(execProgramTempl, p.ISEID)
	if err != nil {
		return err
	}
	if len(resp) != 1 {
		return errors.New("Executing program: Expected exactly one response line")
	}
	if resp[0] != "OK" {
		return fmt.Errorf("Executing program: HM script signals error: %s", resp[0])
	}
	return nil
}

// ReadExecTime reads the last execution time of a ReGaHssProgram.
func (sc *Client) ReadExecTime(p *ProgramDef) (time.Time, error) {
	scriptLog.Debugf("Reading last executing time: %v", p.DisplayName)
	resp, err := sc.ExecuteTempl(readExecTimeTempl, p.ISEID)
	if err != nil {
		return time.Time{}, err
	}
	if len(resp) < 1 {
		return time.Time{}, errors.New("Reading last executing time: Expected at least one response line")
	}
	if resp[0] != "OK" {
		return time.Time{}, fmt.Errorf("Reading last executing time: HM script signals error: %s", resp[0])
	}
	// parse timestamp
	ts, err := time.ParseInLocation("2006-01-02 15:04:05", resp[1], time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("Reading last executing time: Invalid timestamp: %s", resp[1])
	}
	return ts, nil
}

// optFloat64Equal returns true, if both a and b are nil, or *a==*b.
func optFloat64Equal(a *float64, b *float64) bool {
	if (a != nil) != (b != nil) {
		return false
	}
	if (a != nil) && (*a != *b) {
		return false
	}
	return true
}

// optStringEqual returns true, if both a and b are nil, or *a==*b.
func optStringEqual(a *string, b *string) bool {
	if (a != nil) != (b != nil) {
		return false
	}
	if (a != nil) && (*a != *b) {
		return false
	}
	return true
}
