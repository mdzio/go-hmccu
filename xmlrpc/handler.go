package xmlrpc

import (
	"bytes"
	"encoding/xml"
	"github.com/mdzio/go-hmccu/handler"
	"github.com/mdzio/go-hmccu/model"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"

	"github.com/mdzio/go-logging"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/charmap"
)

// max. size of a valid request, if not specified: 10 MB
const requestSizeLimit = 10 * 1024 * 1024

var svrLog = logging.Get("xmlrpc-server")

// Handler implements a http.Handler which can handle XML-RPC requests. Remote
// calls are dispatched to the registered Method's.
type Handler struct {
	handler.BaseHandler
	RequestSizeLimit int64

	mutex   sync.RWMutex
	methods map[string]handler.Method
	unknown func(string, *model.Value) (*model.Value, error)
}

//// Handle registers a Method.
//func (h *Handler) Handle(name string, m Method) {
//	h.mutex.Lock()
//	defer h.mutex.Unlock()
//
//	if h.methods == nil {
//		h.methods = make(map[string]Method)
//	}
//	h.methods[name] = m
//}

//// HandleFunc registers an ordinary function as Method.
//func (h *Handler) HandleFunc(name string, f func(*Value) (*Value, error)) {
//	h.Handle(name, MethodFunc(f))
//}
//
//// HandleUnknownFunc registers an ordinary function to handle unknown methods
//// names.
//func (h *Handler) HandleUnknownFunc(f func(string, *Value) (*Value, error)) {
//	h.mutex.Lock()
//	defer h.mutex.Unlock()
//
//	h.unknown = f
//}
//
//// SystemMethods adds system.multicall and system.listMethods.
//func (h *Handler) SystemMethods() {
//
//	// attention: currently if one methods fails, the complete multicall fails.
//	h.HandleFunc(
//		"system.multicall",
//		func(parameters *Value) (*Value, error) {
//			q := Q(parameters)
//			calls := q.Idx(0).Slice()
//			if q.Err() != nil {
//				return nil, fmt.Errorf("Invalid system.multicall: %v", q.Err())
//			}
//			svrLog.Debugf("Call of method system.multicall with %d elements received", len(calls))
//			var results []*Value
//			for _, call := range calls {
//				methodName := call.Key("methodName").String()
//				// check for an array
//				call.Key("params").Slice()
//				if q.Err() != nil {
//					return nil, fmt.Errorf("Invalid system.multicall: %v", q.Err())
//				}
//				// dispatch call
//				res, err := h.dispatch(methodName, call.Key("params").Value())
//				if err != nil {
//					return nil, fmt.Errorf("Method %s in system.multicall failed: %v", methodName, err)
//				}
//				results = append(results, res)
//			}
//			return &Value{Array: &Array{results}}, nil
//		},
//	)
//
//	h.HandleFunc(
//		"system.listMethods",
//		func(*Value) (*Value, error) {
//			svrLog.Debug("Call of method system.listMethods received")
//			h.mutex.RLock()
//			defer h.mutex.RUnlock()
//
//			names := []*Value{}
//			for name := range h.methods {
//				names = append(names, &Value{FlatString: name})
//			}
//			return &Value{Array: &Array{names}}, nil
//		},
//	)
//}

func (h *Handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	svrLog.Tracef("Request received from %s, URI %s", req.RemoteAddr, req.RequestURI)

	// read request
	limit := h.RequestSizeLimit
	if limit == 0 {
		limit = requestSizeLimit
	}
	reqLimitReader := http.MaxBytesReader(resp, req.Body, limit)
	reqBuf, err := ioutil.ReadAll(reqLimitReader)
	if err != nil {
		svrLog.Errorf("Reading of request failed from %s: %v", req.RemoteAddr, err)
		http.Error(resp, "Reading of request failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	if svrLog.TraceEnabled() {
		// attention: log message is probably ISO8859-1 encoded!
		svrLog.Tracef("Request XML: %s", string(reqBuf))
	}

	// decode request from xml
	reqReader := bytes.NewBuffer(reqBuf)
	methodCall := &MethodCall{}
	dec := xml.NewDecoder(reqReader)
	dec.CharsetReader = charset.NewReaderLabel
	err = dec.Decode(methodCall)
	if err != nil {
		svrLog.Errorf("Decoding of request from %s failed: %v", req.RemoteAddr, err)
		http.Error(resp, "Decoding of request failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	// convert Params to Array
	data := make([]*model.Value, len(methodCall.Params.Param))
	for i, p := range methodCall.Params.Param {
		data[i] = p.Value
	}
	args := &model.Value{
		Array: &model.Array{
			Data: data,
		},
	}

	// dispatch call
	res, err := h.Dispatch(methodCall.MethodName, args)
	var methodResponse *MethodResponse
	if err != nil {
		methodResponse = newFaultResponse(err)
	} else {
		methodResponse = newMethodResponse(res)
	}

	// use ISO8859-1 character encoding for response
	var respBuf bytes.Buffer
	respWriter := charmap.ISO8859_1.NewEncoder().Writer(&respBuf)

	// write xml header
	respWriter.Write([]byte("<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n"))

	// encode response to xml
	enc := xml.NewEncoder(respWriter)
	err = enc.Encode(methodResponse)
	if err != nil {
		svrLog.Errorf("Encoding of response for %s failed: %v", req.RemoteAddr, err)
		http.Error(resp, "Encoding of response failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if svrLog.TraceEnabled() {
		// attention: log message is ISO8859-1 encoded!
		svrLog.Tracef("Response XML: %s", respBuf.String())
	}

	// send response
	resp.Header().Set("Content-Type", "text/xml")
	resp.Header().Set("Content-Length", strconv.Itoa(respBuf.Len()))
	_, err = resp.Write(respBuf.Bytes())
	if err != nil {
		svrLog.Warningf("Sending of response for %s failed: %v", req.RemoteAddr, err)
		return
	}
}

//func (h *Handler) dispatch(methodName string, args *Value) (*Value, error) {
//	h.mutex.RLock()
//	method, ok := h.methods[methodName]
//	unknown := h.unknown
//	h.mutex.RUnlock()
//
//	if !ok {
//		if unknown == nil {
//			unknown = func(name string, _ *Value) (*Value, error) {
//				return nil, fmt.Errorf("Unknown method: %s", name)
//			}
//		}
//		return unknown(methodName, args)
//	}
//	return method.Call(args)
//}
