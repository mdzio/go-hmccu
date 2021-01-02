package handler

import (
	"fmt"
	"github.com/mdzio/go-hmccu/model"
	"github.com/mdzio/go-logging"
	"sync"
)

// max. size of a valid request, if not specified: 10 MB
const requestSizeLimit = 10 * 1024 * 1024

var svrLog = logging.Get("rpc-server")

// A Method is dispatched from a Handler. The argument contains always an array.
type Method interface {
	Call(*model.Value) (*model.Value, error)
}

// MethodFunc is an adapter to use ordinary functions as Method's.
type MethodFunc func(*model.Value) (*model.Value, error)

// Call implements interface Method.
func (m MethodFunc) Call(args *model.Value) (*model.Value, error) {
	return m(args)
}

// BaseHandler implements a base handler use to implement binrpc and xmlrpc handlers.
// Remote calls are dispatched to the registered Method's.
type BaseHandler struct {
	RequestSizeLimit int64

	mutex   sync.RWMutex
	methods map[string]Method
	unknown func(string, *model.Value) (*model.Value, error)
}

// Handle registers a Method.
func (h *BaseHandler) Handle(name string, m Method) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.methods == nil {
		h.methods = make(map[string]Method)
	}
	h.methods[name] = m
}

// HandleFunc registers an ordinary function as Method.
func (h *BaseHandler) HandleFunc(name string, f func(*model.Value) (*model.Value, error)) {
	h.Handle(name, MethodFunc(f))
}

// HandleUnknownFunc registers an ordinary function to handle unknown methods
// names.
func (h *BaseHandler) HandleUnknownFunc(f func(string, *model.Value) (*model.Value, error)) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.unknown = f
}

// SystemMethods adds system.multicall and system.listMethods.
func (h *BaseHandler) SystemMethods() {

	// attention: currently if one methods fails, the complete multicall fails.
	h.HandleFunc(
		"system.multicall",
		func(parameters *model.Value) (*model.Value, error) {
			q := model.Q(parameters)
			calls := q.Idx(0).Slice()
			if q.Err() != nil {
				return nil, fmt.Errorf("Invalid system.multicall: %v", q.Err())
			}
			svrLog.Debugf("Call of method system.multicall with %d elements received", len(calls))
			var results []*model.Value
			for _, call := range calls {
				methodName := call.Key("methodName").String()
				// check for an array
				call.Key("params").Slice()
				if q.Err() != nil {
					return nil, fmt.Errorf("Invalid system.multicall: %v", q.Err())
				}
				// dispatch call
				res, err := h.Dispatch(methodName, call.Key("params").Value())
				if err != nil {
					return nil, fmt.Errorf("Method %s in system.multicall failed: %v", methodName, err)
				}
				results = append(results, res)
			}
			return &model.Value{Array: &model.Array{results}}, nil
		},
	)

	h.HandleFunc(
		"system.listMethods",
		func(*model.Value) (*model.Value, error) {
			svrLog.Debug("Call of method system.listMethods received")
			h.mutex.RLock()
			defer h.mutex.RUnlock()

			names := []*model.Value{}
			for name := range h.methods {
				names = append(names, &model.Value{FlatString: name})
			}
			return &model.Value{Array: &model.Array{names}}, nil
		},
	)
}

func (h *BaseHandler) Dispatch(methodName string, args *model.Value) (*model.Value, error) {
	h.mutex.RLock()
	method, ok := h.methods[methodName]
	unknown := h.unknown
	h.mutex.RUnlock()

	if !ok {
		if unknown == nil {
			unknown = func(name string, _ *model.Value) (*model.Value, error) {
				return nil, fmt.Errorf("Unknown method: %s", name)
			}
		}
		return unknown(methodName, args)
	}
	return method.Call(args)
}
