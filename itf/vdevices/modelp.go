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

// NewBoolParameter creates a BoolParameter (Type: BOOL). For an ACTION parameter
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

// IntParameter represents a HM FLOAT value.
type IntParameter struct {
	Parameter

	// This callback is executed when an external system wants to change the
	// value. Only if this function returns true, the value is actually set.
	OnSetValue func(value int) (ok bool)

	value int
}

// NewIntParameter creates an IntParameter (Type: INTEGER). For an ENUM
// parameter Type must be modified accordingly. The locker of the channel is
// used while modifying the value. Following fields in the parameters
// description are initialized to standard values: Type, Operation, Flags,
// Default (0), Min (-100000), Max (100000), ID.
func NewIntParameter(id string) *IntParameter {
	p := &IntParameter{
		Parameter: Parameter{
			description: &itf.ParameterDescription{
				Type:       itf.ParameterTypeInteger,
				Operations: itf.ParameterOperationRead | itf.ParameterOperationWrite | itf.ParameterOperationEvent,
				Flags:      itf.ParameterFlagVisible,
				Default:    0,
				Max:        1000000000,
				Min:        -1000000000,
				ID:         id,
			},
		},
	}
	p.GenericParameter = p
	return p
}

// SetValue implements interface GenericParameter. This accessor is for external
// systems.
func (p *IntParameter) SetValue(value interface{}) error {
	if p.description.Operations&itf.ParameterOperationWrite == 0 {
		return fmt.Errorf("Parameter not writeable: %s.%s", p.parentDescr.Address, p.description.ID)
	}
	ivalue, ok := value.(int)
	if !ok {
		return fmt.Errorf("Invalid data type for parameter %s.%s: %T", p.parentDescr.Address, p.description.ID, value)
	}
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.OnSetValue == nil {
		ok = true
	} else {
		ok = p.OnSetValue(ivalue)
	}
	if ok {
		p.RawSetValue(ivalue)
	}
	return nil
}

// Value implements interface GenericParameter.  This accessor is for external
// systems.
func (p *IntParameter) Value() interface{} {
	p.locker.Lock()
	defer p.locker.Unlock()
	return p.value
}

// RawSetValue gets called by internal logic. Channel lock is not acquired.
func (p *IntParameter) RawSetValue(value int) {
	p.value = value
	if pub := p.publisher; pub != nil {
		pub.PublishEvent(p.parentDescr.Address, p.description.ID, value)
	}
}

// RawValue gets called by internal logic. Channel lock is not acquired.
func (p *IntParameter) RawValue() int {
	return p.value
}

// FloatParameter represents a HM FLOAT value.
type FloatParameter struct {
	Parameter

	// This callback is executed when an external system wants to change the
	// value. Only if this function returns true, the value is actually set.
	OnSetValue func(value float64) (ok bool)

	value float64
}

// NewFloatParameter creates a FloatParameter (Type: FLOAT). The locker of the
// channel is used while modifying the value. Following fields in the parameters
// description are initialized to standard values: Type, Operation, Flags,
// Default (0.0), Min (-100000), Max (100000), ID.
func NewFloatParameter(id string) *FloatParameter {
	p := &FloatParameter{
		Parameter: Parameter{
			description: &itf.ParameterDescription{
				Type:       itf.ParameterTypeFloat,
				Operations: itf.ParameterOperationRead | itf.ParameterOperationWrite | itf.ParameterOperationEvent,
				Flags:      itf.ParameterFlagVisible,
				Default:    0.0,
				Max:        1000000000.0,
				Min:        -1000000000.0,
				ID:         id,
			},
		},
	}
	p.GenericParameter = p
	return p
}

// SetValue implements interface GenericParameter. This accessor is for external
// systems.
func (p *FloatParameter) SetValue(value interface{}) error {
	if p.description.Operations&itf.ParameterOperationWrite == 0 {
		return fmt.Errorf("Parameter not writeable: %s.%s", p.parentDescr.Address, p.description.ID)
	}
	fvalue, ok := value.(float64)
	if !ok {
		return fmt.Errorf("Invalid data type for parameter %s.%s: %T", p.parentDescr.Address, p.description.ID, value)
	}
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.OnSetValue == nil {
		ok = true
	} else {
		ok = p.OnSetValue(fvalue)
	}
	if ok {
		p.RawSetValue(fvalue)
	}
	return nil
}

// Value implements interface GenericParameter.  This accessor is for external
// systems.
func (p *FloatParameter) Value() interface{} {
	p.locker.Lock()
	defer p.locker.Unlock()
	return p.value
}

// RawSetValue gets called by internal logic. Channel lock is not acquired.
func (p *FloatParameter) RawSetValue(value float64) {
	p.value = value
	if pub := p.publisher; pub != nil {
		pub.PublishEvent(p.parentDescr.Address, p.description.ID, value)
	}
}

// RawValue gets called by internal logic. Channel lock is not acquired.
func (p *FloatParameter) RawValue() float64 {
	return p.value
}

// StringParameter represents a HM STRING value.
type StringParameter struct {
	Parameter

	// This callback is executed when an external system wants to change the
	// value. Only if this function returns true, the value is actually set.
	OnSetValue func(value string) (ok bool)

	value string
}

// NewStringParameter creates a StringParameter (Type: STRING). The locker of
// the channel is used while modifying the value. Following fields in the
// parameters description are initialized to standard values: Type, Operation,
// Flags, Default (""), Min (""), Max (""), ID.
func NewStringParameter(id string) *StringParameter {
	p := &StringParameter{
		Parameter: Parameter{
			description: &itf.ParameterDescription{
				Type:       itf.ParameterTypeString,
				Operations: itf.ParameterOperationRead | itf.ParameterOperationWrite | itf.ParameterOperationEvent,
				Flags:      itf.ParameterFlagVisible,
				Default:    "",
				Max:        "",
				Min:        "",
				ID:         id,
			},
		},
	}
	p.GenericParameter = p
	return p
}

// SetValue implements interface GenericParameter. This accessor is for external
// systems.
func (p *StringParameter) SetValue(value interface{}) error {
	if p.description.Operations&itf.ParameterOperationWrite == 0 {
		return fmt.Errorf("Parameter not writeable: %s.%s", p.parentDescr.Address, p.description.ID)
	}
	svalue, ok := value.(string)
	if !ok {
		return fmt.Errorf("Invalid data type for parameter %s.%s: %T", p.parentDescr.Address, p.description.ID, value)
	}
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.OnSetValue == nil {
		ok = true
	} else {
		ok = p.OnSetValue(svalue)
	}
	if ok {
		p.RawSetValue(svalue)
	}
	return nil
}

// Value implements interface GenericParameter.  This accessor is for external
// systems.
func (p *StringParameter) Value() interface{} {
	p.locker.Lock()
	defer p.locker.Unlock()
	return p.value
}

// RawSetValue gets called by internal logic. Channel lock is not acquired.
func (p *StringParameter) RawSetValue(value string) {
	p.value = value
	if pub := p.publisher; pub != nil {
		pub.PublishEvent(p.parentDescr.Address, p.description.ID, value)
	}
}

// RawValue gets called by internal logic. Channel lock is not acquired.
func (p *StringParameter) RawValue() string {
	return p.value
}
