package binrpc

import (
	"bytes"
	"github.com/mdzio/go-hmccu/handler"
	"github.com/mdzio/go-hmccu/model"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/mdzio/go-logging"
)

// max. size of a valid request, if not specified: 10 MB
const requestSizeLimit = 10 * 1024 * 1024

var svrLog = logging.Get("binrpc-server")

// Handler implements a http.Handler which can handle XML-RPC requests. Remote
// calls are dispatched to the registered Method's.
type Handler struct {
	handler.BaseHandler
	RequestSizeLimit int64

	mutex   sync.RWMutex
	methods map[string]handler.Method
	unknown func(string, *model.Value) (*model.Value, error)
}

func (h *Handler) ServeTCP(conn net.Conn) {
	svrLog.Debug("Request received from %s", conn.RemoteAddr)

	// decode request
	dec := NewDecoder(conn)
	method, params, err := dec.DecodeRequest()
	if err != nil {
		svrLog.Errorf("Decoding of request from %s failed: %v", conn.RemoteAddr, err)
		return
	}

	args := &model.Value{
		Array: &model.Array{
			Data: params,
		},
	}

	svrLog.Debugf("Received method call: %s", method)

	// dispatch call
	res, err := h.Dispatch(method, args)

	buf := bytes.Buffer{}
	e := NewEncoder(&buf)
	if err != nil {
		err := e.EncodeResponse(&model.Value{})
		if err != nil {
			svrLog.Errorf("Failed to encode empty string response: %s", err)
			return
		}
	} else {
		err := e.EncodeResponse(res)
		if err != nil {
			svrLog.Errorf("Failed to encode response: %s", err)
			return
		}

	}
	out, err := ioutil.ReadAll(&buf)
	if err != nil {
		return
	}

	err = conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	if err != nil {
		svrLog.Warningf("Failed to set write deadline: %s", err)
	}

	_, err = conn.Write(out)
	if err != nil {
		svrLog.Warningf("Sending of response for %s failed: %v", conn.RemoteAddr, err)
		return
	}

	//var methodResponse *xmlrpc.MethodResponse
	//if err != nil {
	//	methodResponse = newFaultResponse(err)
	//} else {
	//	methodResponse = newMethodResponse(res)
	//}
	//
	//// use ISO8859-1 character encoding for response
	//var respBuf bytes.Buffer
	//respWriter := charmap.ISO8859_1.NewEncoder().Writer(&respBuf)
	//
	//// write xml header
	//respWriter.Write([]byte("<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n"))
	//
	//// encode response to xml
	//enc := xml.NewEncoder(respWriter)
	//err = enc.EncodeRequest(methodResponse)
	//if err != nil {
	//	svrLog.Errorf("Encoding of response for %s failed: %v", req.RemoteAddr, err)
	//	http.Error(resp, "Encoding of response failed: "+err.Error(), http.StatusInternalServerError)
	//	return
	//}
	//if svrLog.TraceEnabled() {
	//	// attention: log message is ISO8859-1 encoded!
	//	svrLog.Tracef("Response XML: %s", respBuf.String())
	//}
	//
	//// send response
	//resp.Header().Set("Content-Type", "text/xml")
	//resp.Header().Set("Content-Length", strconv.Itoa(respBuf.Len()))
	//_, err = resp.Write(respBuf.Bytes())
	//if err != nil {
	//	svrLog.Warningf("Sending of response for %s failed: %v", req.RemoteAddr, err)
	//	return
	//}
}
