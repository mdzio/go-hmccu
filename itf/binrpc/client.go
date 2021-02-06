package binrpc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"github.com/mdzio/go-logging"
)

var clnLog = logging.Get("binrpc-client")

// Client provides access to an XML-RPC server.
type Client struct {
	Addr string
	// TODO: implement
	//ResponseSizeLimit int64
}

// Call executes an remote procedure call.
func (c *Client) Call(method string, params []*xmlrpc.Value) (*xmlrpc.Value, error) {
	clnLog.Tracef("Calling method %s on %s", method, c.Addr)

	conn, err := net.Dial("tcp", c.Addr)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect: %s", err)
	}

	defer conn.Close()

	buf := bytes.Buffer{}
	e := NewEncoder(&buf)
	err = e.EncodeRequest(method, params)
	if err != nil {
		clnLog.Errorf("Failed to encode request %s: %s", method, err)
		return nil, err
	}
	out, err := ioutil.ReadAll(&buf)
	if err != nil {
		clnLog.Errorf("Failed to read encoded request %s: %s", method, err)
		return nil, err
	}

	_, err = conn.Write(out)
	if err != nil {
		clnLog.Errorf("Failed to send request %s: %s", method, err)
		return nil, err
	}

	err = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		panic(err)
	}
	dec := NewDecoder(conn)
	resp, err := dec.DecodeResponse()
	if err != nil {
		clnLog.Errorf("Failed to decode response %s: %s", method, err)
		return nil, err
	}

	return resp, nil
}
