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
	defer conn.Close()
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

}
