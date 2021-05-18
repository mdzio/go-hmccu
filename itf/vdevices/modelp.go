package vdevices

import (
	"fmt"
	"sync"

	"github.com/mdzio/go-hmccu/itf"
)

// Parameter implements GenericParameter.
type Parameter struct {
	// Only SetValue and Value methods are missing in Parameter.
	GenericParameter

	description *itf.ParameterDescription
	parentDescr *itf.DeviceDescription
	locker      sync.Locker
	publisher   EventPublisher
}

// Description implements interface GenericParameter.
func (p *Parameter) Description() *itf.ParameterDescription {
	return p.description
}

// BoolParameter represents a HM BOOL or ACTION value.
type BoolParameter struct {
	Parameter

	// This callback is executed when an external system wants to change the
	// value. Only if this function returns true, the value is actually set.
	OnSetValue func(value bool) (ok bool)

	value bool
}

// NewBoolParameter creates a BoolParameter (Type: BOOL). For an ACTION paramter
// Type must be modified accordingly. The locker of the channel is used while
// modifying the value. Following fields in the parameters description are
// initialized to standard values: Type, Operation, Flags, Default, Min, Max,
// ID.
func NewBoolParameter(id string) *BoolParameter {
	p := &BoolParameter{
		Parameter: Parameter{
			description: &itf.ParameterDescription{
				Type:       itf.ParameterTypeBool,
				Operations: itf.ParameterOperationRead | itf.ParameterOperationWrite | itf.ParameterOperationEvent,
				Flags:      itf.ParameterFlagVisible,
				Default:    false,
				Max:        true,
				Min:        false,
				ID:         id,
			},
		},
	}
	p.GenericParameter = p
	return p
}

// SetValue implements interface GenericParameter. This accessor is for external
// systems.
func (p *BoolParameter) SetValue(value interface{}) error {
	if p.description.Operations&itf.ParameterOperationWrite == 0 {
		return fmt.Errorf("Parameter not writeable: %s.%s", p.parentDescr.Address, p.description.ID)
	}
	bvalue, ok := value.(bool)
	if !ok {
		return fmt.Errorf("Invalid data type for parameter %s.%s: %T", p.parentDescr.Address, p.description.ID, value)
	}
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.OnSetValue == nil {
		ok = true
	} else {
		ok = p.OnSetValue(bvalue)
	}
	if ok {
		p.RawSetValue(bvalue)
	}
	return nil
}

// Value implements interface GenericParameter.  This accessor is for external
// systems.
func (p *BoolParameter) Value() interface{} {
	p.locker.Lock()
	defer p.locker.Unlock()
	return p.value
}

// RawSetValue gets called by internal logic. Channel lock is not acquired.
func (p *BoolParameter) RawSetValue(value bool) {
	p.value = value
	if pub := p.publisher; pub != nil {
		pub.PublishEvent(p.parentDescr.Address, p.description.ID, value)
	}
}

// RawValue gets called by internal logic. Channel lock is not acquired.
func (p *BoolParameter) RawValue() bool {
	return p.value
}
