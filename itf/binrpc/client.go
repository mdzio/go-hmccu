package binrpc

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"github.com/mdzio/go-logging"
)

const (
	// receive timeout
	receiveTimeout = 15 * time.Second

	// max. size of a valid response, if not specified: 2 MB
	responseSizeLimit = 2 * 1024 * 1024
)

var clnLog = logging.Get("binrpc-client")

// Client provides access to an BIN-RPC server.
type Client struct {
	Addr              string
	ResponseSizeLimit int64
}

// Call executes an remote procedure call. Call implements xmlrpc.Caller.
func (c *Client) Call(method string, params xmlrpc.Values) (*xmlrpc.Value, error) {
	// log
	clnLog.Tracef("Calling method %s on %s with parameters %v", method, c.Addr, params)

	// open connection
	conn, err := net.Dial("tcp", c.Addr)
	if err != nil {
		return nil, fmt.Errorf("Connecting to %s failed: %w", c.Addr, err)
	}
	defer conn.Close()

	// encode request
	buf := bytes.Buffer{}
	e := NewEncoder(&buf)
	err = e.EncodeRequest(method, params)
	if err != nil {
		return nil, fmt.Errorf("Encoding of request for %s failed: %w", c.Addr, err)
	}

	// send request
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("Sending of request to %s failed: %w", c.Addr, err)
	}

	// receive response
	limit := c.ResponseSizeLimit
	if limit == 0 {
		limit = responseSizeLimit
	}
	limitReader := io.LimitReader(conn, limit)
	err = conn.SetReadDeadline(time.Now().Add(receiveTimeout))
	if err != nil {
		return nil, fmt.Errorf("Setting of read deadline failed: %w", err)
	}

	// decode response
	dec := NewDecoder(limitReader)
	resp, err := dec.DecodeResponse()
	if err != nil {
		_, methodError := err.(*xmlrpc.MethodError)
		if !methodError {
			return nil, fmt.Errorf("Decoding of response from %s failed: %w", c.Addr, err)
		}
		clnLog.Tracef("Result: %v", err)
		return nil, err
	}

	// log
	clnLog.Tracef("Result: %v", resp)
	return resp, nil
}
