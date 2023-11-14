package xmlrpc

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"strconv"

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
	RequestSizeLimit int64
	Dispatcher
}

func (h *Handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	svrLog.Tracef("Request received from %s, URI %s", req.RemoteAddr, req.RequestURI)

	// read request
	limit := h.RequestSizeLimit
	if limit == 0 {
		limit = requestSizeLimit
	}
	reqLimitReader := http.MaxBytesReader(resp, req.Body, limit)
	reqBuf, err := io.ReadAll(reqLimitReader)
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
	data := make([]*Value, len(methodCall.Params.Param))
	for i, p := range methodCall.Params.Param {
		data[i] = p.Value
	}
	args := &Value{
		Array: &Array{
			Data: data,
		},
	}

	// dispatch call
	res, err := h.Dispatch(methodCall.MethodName, args)
	var methodResponse *MethodResponse
	if err != nil {
		svrLog.Warningf("Sending error response to %s: %v", req.RemoteAddr, err)
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
