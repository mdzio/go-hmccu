package xmlrpc

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerBadRequest(t *testing.T) {
	h := &Handler{Dispatcher: &Dispatcher{}}
	srv := httptest.NewServer(h)
	defer srv.Close()

	buf := bytes.NewBufferString("invalid request")
	resp, err := http.Post(srv.URL, "text/plain", buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	msg, _ := ioutil.ReadAll(resp.Body)
	if string(msg) != "Decoding of request failed: EOF\n" {
		t.Errorf("unexpected status message: %s", string(msg))
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func TestServerUnknownMethod(t *testing.T) {
	h := &Handler{Dispatcher: &Dispatcher{}}
	srv := httptest.NewServer(h)
	defer srv.Close()

	cln := Client{Addr: srv.URL}

	res, err := cln.Call("unknownMethod", []*Value{})
	if res != nil {
		t.Errorf("unexpected result: %v", res)
	}
	if fault, ok := err.(*MethodError); ok {
		if fault.Code != -1 {
			t.Errorf("unexpected fault code: %d", fault.Code)
		}
		if fault.Message != "Unknown method: unknownMethod" {
			t.Errorf("unexpected fault message: %s", fault.Message)
		}
	} else {
		t.Errorf("invalid error type: %T", err)
	}
}

func TestServer(t *testing.T) {
	h := &Handler{Dispatcher: &Dispatcher{}}
	h.SystemMethods()
	h.HandleFunc("echo", func(args *Value) (*Value, error) {
		q := Q(args)
		if len(q.Slice()) != 1 {
			return nil, errors.New("invalid len")
		}
		return q.Idx(0).Value(), nil
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	cln := Client{Addr: srv.URL}

	resp, err := cln.Call("echo", []*Value{{Int: "123"}})
	if err != nil {
		t.Fatal(err)
	}
	e := Q(resp)
	i := e.Int()
	if e.Err() != nil || i != 123 {
		t.Errorf("unexpected result: %v %d", e.Err(), i)
	}

	resp, err = cln.Call("echo", []*Value{
		{Int: "123"},
		{ElemString: "force error"},
	})
	if resp != nil {
		t.Errorf("unexpected response: %v", resp)
	}
	if fault, ok := err.(*MethodError); ok {
		if fault.Code != -1 || fault.Message != "invalid len" {
			t.Errorf("unexpected error: %v", fault)
		}
	} else {
		t.Errorf("unexpected error type: %T", err)
	}

	resp, err = cln.Call("system.listMethods", []*Value{})
	if err != nil {
		t.Fatal(err)
	}
	e = Q(resp)
	arr := e.Slice()
	if e.Err() != nil {
		t.Fatal(e.Err())
	}
	var methods = make(map[string]bool)
	for _, v := range arr {
		methods[v.String()] = true
	}
	if !(methods["system.multicall"] && methods["system.listMethods"] && methods["echo"]) {
		t.Error("method missing")
	}
}

func TestServerMulticall(t *testing.T) {
	h := &Handler{Dispatcher: &Dispatcher{}}
	h.SystemMethods()
	h.HandleFunc("echo", func(args *Value) (*Value, error) {
		q := Q(args)
		if len(q.Slice()) != 1 {
			return nil, errors.New("invalid len")
		}
		return q.Idx(0).Value(), nil
	})
	srv := httptest.NewServer(h)
	defer srv.Close()
	cln := Client{Addr: srv.URL}

	resp, err := cln.Call("system.multicall", []*Value{
		{
			Array: &Array{
				[]*Value{
					{
						Struct: &Struct{
							[]*Member{
								{
									"methodName",
									&Value{FlatString: "echo"},
								},
								{
									"params",
									&Value{
										Array: &Array{
											[]*Value{
												{
													FlatString: "Hello world!",
												},
											},
										},
									},
								},
							},
						},
					},
					{
						Struct: &Struct{
							[]*Member{
								{
									"methodName",
									&Value{FlatString: "echo"},
								},
								{
									"params",
									&Value{
										Array: &Array{
											[]*Value{
												{
													I4: "123",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	e := Q(resp)
	a := e.Slice()
	if e.Err() != nil {
		t.Error(err)
	}
	if len(a) != 2 {
		t.Fatal("invalid number of results")
	}
	if a[0].String() != "Hello world!" {
		t.Error("invalid first result")
	}
	if a[1].Int() != 123 {
		t.Error("invalid second result")
	}
}

func TestServerWithUnknownMethod(t *testing.T) {
	h := &Handler{Dispatcher: &Dispatcher{}}
	h.HandleUnknownFunc(func(name string, _ *Value) (*Value, error) {
		v, _ := NewValue("Method " + name + " called")
		return v, nil
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	cln := Client{Addr: srv.URL}

	res, err := cln.Call("42", []*Value{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("missing result")
	}
	e := Q(res)
	if str := e.String(); e.Err() != nil || str != "Method 42 called" {
		t.Fatalf("unexpected result: %+v", res)
	}
}
