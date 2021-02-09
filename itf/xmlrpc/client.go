package xmlrpc

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/mdzio/go-logging"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/charmap"
)

// max. size of a valid response, if not specified: 10 MB
const responseSizeLimit = 10 * 1024 * 1024

// Caller is an interface for calling XML-RPC functions.
type Caller interface {
	Call(method string, params Values) (*Value, error)
}

var clnLog = logging.Get("xmlrpc-client")

// Client provides access to an XML-RPC server.
type Client struct {
	Addr              string
	ResponseSizeLimit int64
}

// Call executes an remote procedure call. Call implements Caller.
func (c *Client) Call(method string, params Values) (*Value, error) {
	clnLog.Tracef("Calling method %s on %s", method, c.Addr)

	// build XML object tree
	ps := make([]*Param, len(params))
	for i, p := range params {
		ps[i] = &Param{p}
	}
	methodCall := &MethodCall{
		MethodName: method,
		Params:     &Params{ps},
	}

	// use ISO8859-1 character encoding for request
	var reqBuf bytes.Buffer
	reqWriter := charmap.ISO8859_1.NewEncoder().Writer(&reqBuf)

	// write xml header
	reqWriter.Write([]byte("<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n"))

	// encode request to xml
	enc := xml.NewEncoder(reqWriter)
	err := enc.Encode(methodCall)
	if err != nil {
		return nil, fmt.Errorf("Encoding of request for %s failed: %v", c.Addr, err)
	}
	if clnLog.TraceEnabled() {
		// attention: log message is ISO8859-1 encoded!
		clnLog.Tracef("Request XML: %s", reqBuf.String())
	}

	// http post
	httpResp, err := http.Post(c.Addr, "text/xml", bytes.NewReader(reqBuf.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed on %s: %v", c.Addr, err)
	}
	defer httpResp.Body.Close()

	// check status
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 299 {
		return nil, fmt.Errorf("HTTP request failed on %s with code: %s", c.Addr, httpResp.Status)
	}

	// read response
	limit := c.ResponseSizeLimit
	if limit == 0 {
		limit = responseSizeLimit
	}
	limitReader := io.LimitReader(httpResp.Body, limit)
	respBuf, err := ioutil.ReadAll(limitReader)
	if err != nil {
		return nil, fmt.Errorf("Reading of response failed from %s: %v", c.Addr, err)
	}
	if clnLog.TraceEnabled() {
		// attention: log message is probably ISO8859-1 encoded!
		clnLog.Tracef("Response XML: %s", string(respBuf))
	}

	// decode response from xml
	respReader := bytes.NewBuffer(respBuf)
	resp := &MethodResponse{}
	dec := xml.NewDecoder(respReader)
	dec.CharsetReader = charset.NewReaderLabel
	err = dec.Decode(resp)
	if err != nil {
		return nil, fmt.Errorf("Decoding of response from %s failed: %v", c.Addr, err)
	}

	// check fault
	if resp.Fault != nil {
		e := Q(resp.Fault)
		faultCode := e.Key("faultCode").Int()
		faultString := e.Key("faultString").String()
		if e.Err() != nil {
			return nil, fmt.Errorf("Invalid XML-RPC fault response: %v", e.Err())
		}
		return nil, &MethodError{faultCode, faultString}
	}

	// check response
	if resp.Params == nil || len(resp.Params.Param) != 1 {
		return nil, fmt.Errorf("Invalid or no parameters in response from %s", c.Addr)
	}
	return resp.Params.Param[0].Value, nil
}
