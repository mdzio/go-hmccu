package binrpc

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"github.com/mdzio/go-logging"
)

const (
	// receive timeout
	sendTimeout = 15 * time.Second

	// max. size of a valid request, if not specified: 2 MB
	requestSizeLimit = 2 * 1024 * 1024
)

var svrLog = logging.Get("binrpc-server")

// Server is a BIN-RPC server.
type Server struct {
	*xmlrpc.Dispatcher
	Addr             string
	ServeErr         chan<- error
	RequestSizeLimit int64

	listener net.Listener
	stop     chan struct{}
	done     chan struct{}
}

// Start starts the TCP server for handling BIN-RPC requests.
func (s *Server) Start() error {
	if s.RequestSizeLimit == 0 {
		s.RequestSizeLimit = requestSizeLimit
	}
	// avoid blocking
	s.stop = make(chan struct{}, 1)
	s.done = make(chan struct{}, 1)

	// start listening
	svrLog.Infof("Starting BIN-RPC server on address %s", s.Addr)
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("Listen on address %s failed: %w", s.Addr, err)
	}
	s.listener = l

	// start serving
	var delay time.Duration
	go func() {
		defer s.listener.Close()
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				// stop request?
				select {
				case <-s.stop:
					// signal server is down
					s.done <- struct{}{}
					return
				default:
				}
				// temporary error?
				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					// sleep on accept failure
					if delay == 0 {
						delay = 5 * time.Millisecond
					} else {
						delay *= 2
					}
					if max := 1 * time.Second; delay > max {
						delay = max
					}
					svrLog.Tracef("Accept failed: %v", err)
					time.Sleep(delay)
					// retry
					continue
				}
				// signal server is down
				s.done <- struct{}{}
				// signal error
				s.ServeErr <- err
				return
			}
			// handle connection
			go s.handle(conn)
		}
	}()
	return nil
}

// Stop stops the TCP server.
func (s *Server) Stop() {
	svrLog.Debug("Shutting down BIN-RPC server")
	s.stop <- struct{}{}
	s.listener.Close()
	<-s.done
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	svrLog.Trace("Request received from ", conn.RemoteAddr())

	// decode request
	dec := NewDecoder(conn)
	method, params, err := dec.DecodeRequest()
	if err != nil {
		svrLog.Errorf("Decoding of request from %s failed: %w", conn.RemoteAddr(), err)
		return
	}
	svrLog.Debugf("Received call from %s of method %s with parameters %s", method, conn.RemoteAddr(), params)

	// repack params as xmlrpc.Array
	args := &xmlrpc.Value{
		Array: &xmlrpc.Array{
			Data: params,
		},
	}

	// dispatch call
	res, merr := s.Dispatch(method, args)

	// encode response
	buf := bytes.Buffer{}
	e := NewEncoder(&buf)
	// method error?
	if merr != nil {
		// encode fault response
		err := e.EncodeFault(merr)
		if err != nil {
			svrLog.Errorf("Encoding of fault response %v failed: %v", merr, err)
			return
		}
		svrLog.Debugf("Sending response to %s: %v", conn.RemoteAddr(), merr)
	} else {
		// encode method result
		err := e.EncodeResponse(res)
		if err != nil {
			svrLog.Errorf("Encoding of response %v failed: %v", res, err)
			return
		}
		svrLog.Debugf("Sending response to %s: %v", conn.RemoteAddr(), res)
	}

	// send response
	err = conn.SetWriteDeadline(time.Now().Add(sendTimeout))
	if err != nil {
		svrLog.Warningf("Setting of timeout for sending failed: %v", err)
	}
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		svrLog.Warningf("Sending of response for %s failed: %v", conn.RemoteAddr(), err)
		return
	}
}
