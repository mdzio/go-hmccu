package binrpc

import (
	"errors"
	"log"
	"testing"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
)

func TestServer(t *testing.T) {
	// setup server
	serr := make(chan error)
	svr := &Server{
		Addr:       ":2123",
		ServeErr:   serr,
		Dispatcher: &xmlrpc.BasicDispatcher{},
	}
	svr.AddSystemMethods()
	svr.HandleFunc("echo", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 1 {
			return nil, errors.New("invalid len")
		}
		return q.Idx(0).Value(), nil
	})

	// start server
	err := svr.Start()
	if err != nil {
		log.Fatal(err)
	}
	defer svr.Stop()

	// create client
	cln := Client{Addr: "127.0.0.1:2123"}

	// test 1
	resp, err := cln.Call("echo", []*xmlrpc.Value{{Int: "123"}})
	if err != nil {
		t.Fatal(err)
	}
	e := xmlrpc.Q(resp)
	i := e.Int()
	if e.Err() != nil || i != 123 {
		t.Errorf("unexpected result: %v %d", e.Err(), i)
	}

	// test 2
	resp, err = cln.Call("echo", xmlrpc.Values{
		{Int: "123"},
		{ElemString: "force error"},
	})
	if resp != nil {
		t.Errorf("unexpected response: %v", resp)
	}
	if fault, ok := err.(*xmlrpc.MethodError); ok {
		if fault.Code != -1 || fault.Message != "invalid len" {
			t.Errorf("unexpected error: %v", fault)
		}
	} else {
		t.Errorf("unexpected error type: %T", err)
	}

	// expect no serve error
	select {
	case err = <-serr:
	default:
		err = nil
	}
	if err != nil {
		t.Error(err)
	}
}
