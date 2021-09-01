package xmlrpc

import (
	"bytes"
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/mdzio/go-logging"
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
	reqBuf, err := ioutil.ReadAll(reqLimitReader)
	if err != nil {
		svrLog.Errorf("Reading of request failed from %s: %v", req.RemoteAddr, err)
		http.Error(resp, "Reading of request failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	if svrLog.TraceEnabled() {
		svrLog.Tracef("Request XML: %s", string(reqBuf))
	}

	// decode request from xml
	reqReader := bytes.NewBuffer(reqBuf)
	methodCall := &MethodCall{}
	dec := xml.NewDecoder(reqReader)
	// override wrong XML encoding attribute (ISO-8859-1) from ReGaHss, request
	// is already encoded in UTF-8
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}
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
		methodResponse = newFaultResponse(err)
	} else {
		methodResponse = newMethodResponse(res)
	}

	// write xml header, use standard UTF-8 character encoding for response
	var respBuf bytes.Buffer
	respBuf.Write([]byte("<?xml version=\"1.0\"?>\n"))

	// encode response to xml
	enc := xml.NewEncoder(&respBuf)
	err = enc.Encode(methodResponse)
	if err != nil {
		svrLog.Errorf("Encoding of response for %s failed: %v", req.RemoteAddr, err)
		http.Error(resp, "Encoding of response failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if svrLog.TraceEnabled() {
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
