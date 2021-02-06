package binrpc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"github.com/mdzio/go-logging"
)

var svrLog = logging.Get("binrpc-server")

// Server starts a TCP server for handling BIN-RPC requests.
func Server(addr string, handler *Handler) {
	listenAddr := strings.Replace(addr, "xmlrpc_bin://", "", 1)
	l, err := net.Listen("tcp4", listenAddr)
	if err != nil {
		// TODO: use logging
		fmt.Println(err)
		return
	}
	defer l.Close()
	rand.Seed(time.Now().Unix())

	for {
		c, err := l.Accept()
		if err != nil {
			// TODO: use logging
			fmt.Println(err)
			return
		}
		go handler.ServeTCP(c)
	}
}

// Handler implements a http.Handler which can handle XML-RPC requests. Remote
// calls are dispatched to the registered Method's.
type Handler struct {
	xmlrpc.Dispatcher
	// TODO: implement
	//RequestSizeLimit int64
}

// ServeTCP serves an incoming TCP connection.
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

	args := &xmlrpc.Value{
		Array: &xmlrpc.Array{
			Data: params,
		},
	}

	svrLog.Debugf("Received method call: %s", method)

	// dispatch call
	res, err := h.Dispatch(method, args)

	buf := bytes.Buffer{}
	e := NewEncoder(&buf)
	if err != nil {
		err := e.EncodeResponse(&xmlrpc.Value{})
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
