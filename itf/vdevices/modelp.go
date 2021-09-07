package vdevices

import (
	"fmt"

	"github.com/mdzio/go-hmccu/itf"
)

// Parameter implements GenericParameter.
type Parameter struct {
	description *itf.ParameterDescription
	parentDescr *itf.DeviceDescription
	publisher   EventPublisher
}

// SetParentDescr implements interface GenericParameter.
func (p *Parameter) SetParentDescr(parentDescr *itf.DeviceDescription) {
	p.parentDescr = parentDescr
}

// SetPublisher implements interface GenericParameter.
func (p *Parameter) SetPublisher(publisher EventPublisher) {
	p.publisher = publisher
}

// Description implements interface GenericParameter.
func (p *Parameter) Description() *itf.ParameterDescription {
	return p.description
}

// Description implements interface GenericParameter.
func (p *Parameter) publishValue(value interface{}) {
	// updates of master params are not published
	if pub := p.publisher; pub != nil {
		pub.PublishEvent(p.parentDescr.Address, p.description.ID, value)
	}
}

// BoolParameter represents a HM BOOL or ACTION value.
type BoolParameter struct {
	Parameter

	// This callback is executed when an external system wants to change the
	// value. Only if this function returns true, the value is actually set. The
	// device/channel is locked.
	OnSetValue func(value bool) (ok bool)

	value bool
}

// check interface implementation
var _ GenericParameter = (*BoolParameter)(nil)

// NewBoolParameter creates a BoolParameter (Type: BOOL). For an ACTION parameter
// Type must be modified accordingly. The locker of the channel is used while
// modifying the value. Following fields in the parameters description are
// initialized to standard values: Type, Operation, Flags, Default, Min, Max,
// ID.
func NewBoolParameter(id string) *BoolParameter {
	return &BoolParameter{
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
	if p.OnSetValue == nil || p.OnSetValue(bvalue) {
		p.publishValue(bvalue)
		p.value = bvalue
	}
	return nil
}

// InternalSetValue implements ValueAccessor.
func (p *BoolParameter) InternalSetValue(value interface{}) error {
	bvalue, ok := value.(bool)
	if !ok {
		return fmt.Errorf("Invalid data type for parameter %s.%s: %T", p.parentDescr.Address, p.description.ID, value)
	}
	p.publishValue(bvalue)
	p.value = bvalue
	return nil
}

// Value implements interface GenericParameter.  This accessor is for external
// systems.
func (p *BoolParameter) Value() interface{} {
	return p.value
}

// IntParameter represents a HM FLOAT value.
type IntParameter struct {
	Parameter

	// This callback is executed when an external system wants to change the
	// value. Only if this function returns true, the value is actually set. The
	// device/channel is locked.
	OnSetValue func(value int) (ok bool)

	value int
}

// check interface implementation
var _ GenericParameter = (*IntParameter)(nil)

// NewIntParameter creates an IntParameter (Type: INTEGER). For an ENUM
// parameter Type must be modified accordingly. The locker of the channel is
// used while modifying the value. Following fields in the parameters
// description are initialized to standard values: Type, Operation, Flags,
// Default (0), Min (-100000), Max (100000), ID.
func NewIntParameter(id string) *IntParameter {
	return &IntParameter{
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
}

func (p *IntParameter) toInt(value interface{}) (int, error) {
	ivalue, ok := value.(int)
	if !ok {
		// accept float64 as well
		fvalue, fok := value.(float64)
		if fok {
			ivalue = int(fvalue)
			// accept only integer numbers
			ok = float64(ivalue) == fvalue
		}
		if !ok {
			return 0, fmt.Errorf("Invalid data type for parameter %s.%s: %T", p.parentDescr.Address, p.description.ID, value)
		}
	}
	// check range only for ENUM
	if p.Description().Type == itf.ParameterTypeEnum {
		min, ok := p.Description().Min.(int)
		if ok && ivalue < min {
			return 0, fmt.Errorf("Value below minimum for parameter %s.%s: %v", p.parentDescr.Address, p.description.ID, ivalue)
		}
		max, ok := p.Description().Max.(int)
		if ok && ivalue > max {
			return 0, fmt.Errorf("Value above maximum for parameter %s.%s: %v", p.parentDescr.Address, p.description.ID, ivalue)
		}
	}
	return ivalue, nil
}

// SetValue implements interface GenericParameter. This accessor is for external
// systems.
func (p *IntParameter) SetValue(value interface{}) error {
	if p.description.Operations&itf.ParameterOperationWrite == 0 {
		return fmt.Errorf("Parameter not writeable: %s.%s", p.parentDescr.Address, p.description.ID)
	}
	ivalue, err := p.toInt(value)
	if err != nil {
		return err
	}
	if p.OnSetValue == nil || p.OnSetValue(ivalue) {
		p.publishValue(ivalue)
		p.value = ivalue
	}
	return nil
}

// InternalSetValue implements ValueAccessor.
func (p *IntParameter) InternalSetValue(value interface{}) error {
	ivalue, err := p.toInt(value)
	if err != nil {
		return err
	}
	p.publishValue(ivalue)
	p.value = ivalue
	return nil
}

// Value implements interface GenericParameter.  This accessor is for external
// systems.
func (p *IntParameter) Value() interface{} {
	return p.value
}

// FloatParameter represents a HM FLOAT value.
type FloatParameter struct {
	Parameter

	// This callback is executed when an external system wants to change the
	// value. Only if this function returns true, the value is actually set. The
	// device/channel is locked.
	OnSetValue func(value float64) (ok bool)

	value float64
}

// check interface implementation
var _ GenericParameter = (*FloatParameter)(nil)

// NewFloatParameter creates a FloatParameter (Type: FLOAT). The locker of the
// channel is used while modifying the value. Following fields in the parameters
// description are initialized to standard values: Type, Operation, Flags,
// Default (0.0), Min (-100000), Max (100000), ID.
func NewFloatParameter(id string) *FloatParameter {
	return &FloatParameter{
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
	if p.OnSetValue == nil || p.OnSetValue(fvalue) {
		p.publishValue(fvalue)
		p.value = fvalue
	}
	return nil
}

// InternalSetValue implements ValueAccessor.
func (p *FloatParameter) InternalSetValue(value interface{}) error {
	fvalue, ok := value.(float64)
	if !ok {
		return fmt.Errorf("Invalid data type for parameter %s.%s: %T", p.parentDescr.Address, p.description.ID, value)
	}
	p.publishValue(fvalue)
	p.value = fvalue
	return nil
}

// Value implements interface GenericParameter.  This accessor is for external
// systems.
func (p *FloatParameter) Value() interface{} {
	return p.value
}

// StringParameter represents a HM STRING value.
type StringParameter struct {
	Parameter

	// This callback is executed when an external system wants to change the
	// value. Only if this function returns true, the value is actually set. The
	// device/channel is locked.
	OnSetValue func(value string) (ok bool)

	value string
}

// check interface implementation
var _ GenericParameter = (*StringParameter)(nil)

// NewStringParameter creates a StringParameter (Type: STRING). The locker of
// the channel is used while modifying the value. Following fields in the
// parameters description are initialized to standard values: Type, Operation,
// Flags, Default (""), Min (""), Max (""), ID.
func NewStringParameter(id string) *StringParameter {
	return &StringParameter{
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
	if p.OnSetValue == nil || p.OnSetValue(svalue) {
		p.publishValue(svalue)
		p.value = svalue
	}
	return nil
}

// InternalSetValue implements ValueAccessor.
func (p *StringParameter) InternalSetValue(value interface{}) error {
	svalue, ok := value.(string)
	if !ok {
		return fmt.Errorf("Invalid data type for parameter %s.%s: %T", p.parentDescr.Address, p.description.ID, value)
	}
	p.publishValue(svalue)
	p.value = svalue
	return nil
}

// Value implements interface GenericParameter.  This accessor is for external
// systems.
func (p *StringParameter) Value() interface{} {
	return p.value
}
