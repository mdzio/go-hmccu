package xmlrpc

import (
	"fmt"
	"sync"
)

// Dispatcher dispatches a received XML-RPC call to registered handlers.
type Dispatcher interface {
	AddSystemMethods()
	Handle(name string, m Method)
	HandleFunc(name string, f func(*Value) (*Value, error))
	HandleUnknownFunc(f func(string, *Value) (*Value, error))
	Dispatch(methodName string, args *Value) (*Value, error)
}

// BasicDispatcher dispatches an XML-RPC call to a registered function.
type BasicDispatcher struct {
	mutex   sync.RWMutex
	methods map[string]Method
	unknown func(string, *Value) (*Value, error)
}

// A Method is dispatched from a Handler. The argument contains always an array.
type Method interface {
	Call(*Value) (*Value, error)
}

// MethodFunc is an adapter to use ordinary functions as Method's.
type MethodFunc func(*Value) (*Value, error)

// Call implements interface Method.
func (m MethodFunc) Call(args *Value) (*Value, error) {
	return m(args)
}

// Handle registers a Method.
func (d *BasicDispatcher) Handle(name string, m Method) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.methods == nil {
		d.methods = make(map[string]Method)
	}
	d.methods[name] = m
}

// HandleFunc registers an ordinary function as Method.
func (d *BasicDispatcher) HandleFunc(name string, f func(*Value) (*Value, error)) {
	d.Handle(name, MethodFunc(f))
}

// HandleUnknownFunc registers an ordinary function to handle unknown methods
// names.
func (d *BasicDispatcher) HandleUnknownFunc(f func(string, *Value) (*Value, error)) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.unknown = f
}

// AddSystemMethods adds system.multicall and system.listMethods.
func (d *BasicDispatcher) AddSystemMethods() {

	// attention: currently if one methods fails, the complete multicall fails.
	d.HandleFunc(
		"system.multicall",
		func(parameters *Value) (*Value, error) {
			q := Q(parameters)
			calls := q.Idx(0).Slice()
			if q.Err() != nil {
				return nil, fmt.Errorf("Invalid system.multicall: %v", q.Err())
			}
			svrLog.Debugf("Call of method system.multicall with %d elements received", len(calls))
			var results []*Value
			for _, call := range calls {
				methodName := call.Key("methodName").String()
				// check for an array
				call.Key("params").Slice()
				if q.Err() != nil {
					return nil, fmt.Errorf("Invalid system.multicall: %v", q.Err())
				}
				// dispatch call
				res, err := d.Dispatch(methodName, call.Key("params").Value())
				if err != nil {
					return nil, fmt.Errorf("Method %s in system.multicall failed: %v", methodName, err)
				}
				results = append(results, res)
			}
			return &Value{Array: &Array{results}}, nil
		},
	)

	d.HandleFunc(
		"system.listMethods",
		func(*Value) (*Value, error) {
			svrLog.Debug("Call of method system.listMethods received")
			d.mutex.RLock()
			defer d.mutex.RUnlock()

			names := []*Value{}
			for name := range d.methods {
				names = append(names, &Value{FlatString: name})
			}
			return &Value{Array: &Array{names}}, nil
		},
	)

	// attention: This implementation returns always an empty string.
	d.HandleFunc(
		"system.methodHelp",
		func(*Value) (*Value, error) {
			svrLog.Debug("Call of method system.methodHelp received")
			return &Value{}, nil
		},
	)
}

// Dispatch dispatches a method call to a registered function.
func (d *BasicDispatcher) Dispatch(methodName string, args *Value) (*Value, error) {
	d.mutex.RLock()
	method, ok := d.methods[methodName]
	unknown := d.unknown
	d.mutex.RUnlock()

	if !ok {
		if unknown == nil {
			unknown = func(name string, _ *Value) (*Value, error) {
				return nil, fmt.Errorf("Unknown method: %s", name)
			}
		}
		return unknown(methodName, args)
	}
	return method.Call(args)
}
