package vdevices

import (
	"github.com/mdzio/go-hmccu/itf"
)

// addInstallTest adds the INSTALL_TEST parameter for simulating a channel/device test
func addInstallTest(ch GenericChannel) {
	p := NewBoolParameter("INSTALL_TEST")
	p.description.Type = itf.ParameterTypeAction
	p.description.Operations = itf.ParameterOperationWrite
	p.description.Flags = itf.ParameterFlagVisible | itf.ParameterFlagInternal
	ch.AddValueParam(p)
}

// MaintenanceChannel is a standard HM device maintenance channel. The first
// channel (Index: 0) of every HM device should be a maintenance channel.
type MaintenanceChannel struct {
	Channel

	unreach       *BoolParameter
	stickyUnreach *BoolParameter
}

// NewMaintenanceChannel creates a new maintenance channel and adds it to the
// device.
func NewMaintenanceChannel(device *Device) *MaintenanceChannel {
	c := new(MaintenanceChannel)
	c.Channel.Init("MAINTENANCE")
	c.description.Flags = itf.DeviceFlagVisible | itf.DeviceFlagInternal
	// adding channel to device also initializes some fields
	device.AddChannel(c)
	addInstallTest(c)

	// add UNREACH parameter
	c.unreach = NewBoolParameter("UNREACH")
	c.unreach.description.Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.unreach.description.Flags = itf.ParameterFlagVisible | itf.ParameterFlagService
	c.AddValueParam(c.unreach)

	// add STICKY_UNREACH parameter
	c.stickyUnreach = NewBoolParameter("STICKY_UNREACH")
	c.stickyUnreach.description.Operations = itf.ParameterOperationRead | itf.ParameterOperationWrite | itf.ParameterOperationEvent
	c.stickyUnreach.description.Flags = itf.ParameterFlagVisible | itf.ParameterFlagService | itf.ParameterFlagSticky
	c.AddValueParam(c.stickyUnreach)
	return c
}

// SetUnreach sets the connection state of the device.
func (c *MaintenanceChannel) SetUnreach(value bool) {
	if value && !c.unreach.Value().(bool) {
		c.stickyUnreach.InternalSetValue(true)
	}
	c.unreach.InternalSetValue(value)
}

// DigitalChannel implements a standard HM switch channel.
type DigitalChannel struct {
	Channel

	// This callback is executed when an external system wants to change the
	// state. Only if this function returns true, the state is actually set.
	OnSetState func(value bool) (ok bool)

	state *BoolParameter
}

// NewDigitalChannel creates a new HM digital channel and adds it to the device.
// The field OnSetState must be set to be able to react to external value
// changes.
func NewDigitalChannel(device *Device, channelType, control string) *DigitalChannel {
	c := new(DigitalChannel)
	c.Channel.Init(channelType)
	// adding channel to device also initializes some fields
	device.AddChannel(&c.Channel)
	addInstallTest(&c.Channel)

	// add STATE parameter
	c.state = NewBoolParameter("STATE")
	c.state.description.Control = control
	c.state.OnSetValue = func(value bool) bool {
		if c.OnSetState != nil {
			return c.OnSetState(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.state)
	return c
}

// SetState sets the state of the switch.
func (c *DigitalChannel) SetState(value bool) {
	c.state.InternalSetValue(value)
}

// State returns the state of the switch.
func (c *DigitalChannel) State() bool {
	return c.state.Value().(bool)
}

// NewSwitchChannel creates a new HM switch channel and adds it to the device.
// The field OnSetState must be set to be able to react to external value
// changes.
func NewSwitchChannel(device *Device) *DigitalChannel {
	return NewDigitalChannel(device, "SWITCH", "SWITCH.STATE")
}

// NewDoorSensorChannel creates a new HM door sensor channel and adds it to the
// device. The field OnSetState must be set to be able to react to external
// value changes.
func NewDoorSensorChannel(device *Device) *DigitalChannel {
	return NewDigitalChannel(device, "SHUTTER_CONTACT", "DOOR_SENSOR.STATE")
}

// KeyChannel implements a standard HM key channel.
type KeyChannel struct {
	Channel
	OnPressShort func() bool
	OnPressLong  func() bool

	pressShort *BoolParameter
	pressLong  *BoolParameter
}

// NewKeyChannel creates a new HM key channel and adds it to the device.
func NewKeyChannel(device *Device) *KeyChannel {
	c := new(KeyChannel)
	c.Channel.Init("KEY_TRANSCEIVER")
	// adding channel to device also initializes some fields
	device.AddChannel(&c.Channel)
	addInstallTest(&c.Channel)

	// add PRESS_SHORT parameter
	c.pressShort = NewBoolParameter("PRESS_SHORT")
	c.pressShort.description.Type = itf.ParameterTypeAction
	c.pressShort.description.Operations = itf.ParameterOperationWrite | itf.ParameterOperationEvent
	c.pressShort.description.Control = "BUTTON.SHORT"
	c.pressShort.OnSetValue = func(value bool) bool {
		if c.OnPressShort != nil {
			return c.OnPressShort()
		} else {
			return true
		}
	}
	c.AddValueParam(c.pressShort)

	// add PRESS_LONG parameter
	c.pressLong = NewBoolParameter("PRESS_LONG")
	c.pressLong.description.Type = itf.ParameterTypeAction
	c.pressLong.description.Operations = itf.ParameterOperationWrite | itf.ParameterOperationEvent
	c.pressLong.description.Control = "BUTTON.LONG"
	c.pressLong.OnSetValue = func(value bool) bool {
		if c.OnPressLong != nil {
			return c.OnPressLong()
		} else {
			return true
		}
	}
	c.AddValueParam(c.pressLong)
	return c
}

// PressShort sends a press short event.
func (c *KeyChannel) PressShort() {
	c.pressShort.InternalSetValue(true)
}

// PressShort sends a press long event.
func (c *KeyChannel) PressLong() {
	c.pressLong.InternalSetValue(true)
}

// AnalogInputChannel implements a HM analog input channel (e.g.
// HmIP-MIO16-PCB:1).
type AnalogInputChannel struct {
	Channel

	// These callbacks are executed when an external system wants to change the
	// values. Only if the function returns true, the value is actually set.
	OnSetVoltage       func(value float64) (ok bool)
	OnSetVoltageStatus func(value int) (ok bool)

	voltage       *FloatParameter
	voltageStatus *IntParameter
}

// NewAnalogInputChannel creates a new HM analog input channel and adds it to the device.
// The field OnSetVoltage must be set to be able to react to external value
// changes.
func NewAnalogInputChannel(device *Device) *AnalogInputChannel {
	c := new(AnalogInputChannel)
	c.Channel.Init("ANALOG_INPUT_TRANSMITTER")
	// adding channel to device also initializes some fields
	device.AddChannel(&c.Channel)
	addInstallTest(&c.Channel)

	// add VOLTAGE parameter
	c.voltage = NewFloatParameter("VOLTAGE")
	c.voltage.description.Control = "ANALOG_INPUT.VOLTAGE"
	c.voltage.OnSetValue = func(value float64) bool {
		if c.OnSetVoltage != nil {
			return c.OnSetVoltage(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.voltage)

	// add VOLTAGE_STATUS parameter
	c.voltageStatus = NewIntParameter("VOLTAGE_STATUS")
	c.voltageStatus.description.Type = itf.ParameterTypeEnum
	c.voltageStatus.description.Control = "ANALOG_INPUT.VOLTAGE_STATUS"
	c.voltageStatus.description.Default = "NORMAL"
	c.voltageStatus.description.Min = "NORMAL"
	c.voltageStatus.description.Max = "OVERFLOW"
	c.voltageStatus.description.ValueList = []string{"NORMAL", "UNKNOWN", "OVERFLOW"}
	c.voltageStatus.OnSetValue = func(value int) bool {
		if c.OnSetVoltage != nil {
			return c.OnSetVoltageStatus(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.voltageStatus)
	return c
}

// SetVoltage sets the voltage of the analog input.
func (c *AnalogInputChannel) SetVoltage(value float64) {
	c.voltage.InternalSetValue(value)
}

// Voltage returns the voltage of the analog input.
func (c *AnalogInputChannel) Voltage() float64 {
	return c.voltage.Value().(float64)
}

// SetVoltageStatus sets the voltage status of the analog input.
func (c *AnalogInputChannel) SetVoltageStatus(value int) {
	c.voltageStatus.InternalSetValue(value)
}

// VoltageStatus returns the voltage status of the analog input.
func (c *AnalogInputChannel) VoltageStatus() int {
	return c.voltageStatus.Value().(int)
}

// DimmerChannel implements a HM dimmer channel (e.g. HM-LC-Dim1TPBU-FM:1).
type DimmerChannel struct {
	Channel

	// These callbacks are executed when an external system wants to change the
	// values. Only if the function returns true, the value is actually set.
	OnSetLevel    func(value float64) (ok bool)
	OnSetOldLevel func() (ok bool)
	OnSetRampTime func(value float64) (ok bool)
	OnSetOnTime   func(value float64) (ok bool)

	level    *FloatParameter
	oldLevel *BoolParameter
	rampTime *FloatParameter
	onTime   *FloatParameter
	working  *BoolParameter
}

// NewDimmerChannel creates a new HM dimmer channel and adds it to the device.
func NewDimmerChannel(device *Device) *DimmerChannel {
	c := new(DimmerChannel)
	c.Channel.Init("DIMMER")
	// adding channel to device also initializes some fields
	device.AddChannel(&c.Channel)
	addInstallTest(&c.Channel)

	// add LEVEL parameter
	c.level = NewFloatParameter("LEVEL")
	c.level.description.Control = "DIMMER.LEVEL"
	c.level.description.TabOrder = 0
	c.level.description.Default = 0.0
	c.level.description.Min = 0.0
	c.level.description.Max = 1.0
	c.level.description.Unit = "100%"
	c.level.OnSetValue = func(value float64) bool {
		if c.OnSetLevel != nil {
			return c.OnSetLevel(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.level)

	// add OLD_LEVEL parameter
	c.oldLevel = NewBoolParameter("OLD_LEVEL")
	c.oldLevel.description.Control = "DIMMER.OLD_LEVEL"
	c.oldLevel.description.TabOrder = 1
	c.oldLevel.description.Type = itf.ParameterTypeAction
	c.oldLevel.description.Operations = itf.ParameterOperationWrite
	c.oldLevel.OnSetValue = func(value bool) bool {
		if c.OnSetOldLevel != nil {
			return c.OnSetOldLevel()
		} else {
			return true
		}
	}
	c.AddValueParam(c.oldLevel)

	// add RAMP_TIME parameter
	c.rampTime = NewFloatParameter("RAMP_TIME")
	c.rampTime.description.Operations = itf.ParameterOperationWrite
	c.rampTime.description.Control = "NONE"
	c.rampTime.description.TabOrder = 2
	// set default value
	c.rampTime.description.Default = 0.5
	c.rampTime.value.Store(float64(0.5))
	c.rampTime.description.Min = 0.0
	c.rampTime.description.Max = 8.58259456e+07
	c.rampTime.description.Unit = "s"
	c.rampTime.OnSetValue = func(value float64) bool {
		if c.OnSetRampTime != nil {
			return c.OnSetRampTime(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.rampTime)

	// add ON_TIME parameter
	c.onTime = NewFloatParameter("ON_TIME")
	c.onTime.description.Operations = itf.ParameterOperationWrite
	c.onTime.description.Control = "NONE"
	c.onTime.description.TabOrder = 3
	// set default value
	c.onTime.description.Default = 0.5
	c.onTime.value.Store(float64(0.5))
	c.onTime.description.Min = 0.0
	c.onTime.description.Max = 8.58259456e+07
	c.onTime.description.Unit = "s"
	c.onTime.OnSetValue = func(value float64) bool {
		if c.OnSetOnTime != nil {
			return c.OnSetOnTime(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.onTime)

	// add WORKING parameter
	c.working = NewBoolParameter("WORKING")
	c.working.description.Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.working.description.Flags = itf.ParameterFlagVisible | itf.ParameterFlagInternal
	c.working.description.TabOrder = 4
	c.AddValueParam(c.working)

	return c
}

// SetLevel sets the level of the dimmer.
func (c *DimmerChannel) SetLevel(value float64) {
	c.level.InternalSetValue(value)
}

// Level returns the level of the dimmer.
func (c *DimmerChannel) Level() float64 {
	return c.level.Value().(float64)
}

// SetRampTime sets the ramp time of the dimmer.
func (c *DimmerChannel) SetRampTime(value float64) {
	c.rampTime.InternalSetValue(value)
}

// RampTime returns the ramp time of the dimmer.
func (c *DimmerChannel) RampTime() float64 {
	return c.rampTime.Value().(float64)
}

// SetOnTime sets the on time of the dimmer.
func (c *DimmerChannel) SetOnTime(value float64) {
	c.onTime.InternalSetValue(value)
}

// OnTime returns the on time of the dimmer.
func (c *DimmerChannel) OnTime() float64 {
	return c.onTime.Value().(float64)
}

// SetWorking sets working state of the dimmer.
func (c *DimmerChannel) SetWorking(value bool) {
	c.working.InternalSetValue(value)
}

// Working returns the working state of the dimmer.
func (c *DimmerChannel) Working() bool {
	return c.working.Value().(bool)
}

// TemperatureChannel implements a HM temperature channel (e.g. HmIP-STHO:1).
type TemperatureChannel struct {
	Channel

	// These callbacks are executed when an external system wants to change the
	// values. Only if the function returns true, the value is actually set.
	OnSetTemperature       func(value float64) (ok bool)
	OnSetTemperatureStatus func(value int) (ok bool)
	OnSetHumidity          func(value int) (ok bool)
	OnSetHumidityStatus    func(value int) (ok bool)

	temperature       *FloatParameter
	temperatureStatus *IntParameter
	humidity          *IntParameter
	humidityStatus    *IntParameter
}

// NewTemperatureChannel creates a new HM temperature channel and adds it to the device.
func NewTemperatureChannel(device *Device) *TemperatureChannel {
	c := new(TemperatureChannel)
	c.Channel.Init("CLIMATE_TRANSCEIVER")
	// adding channel to device also initializes some fields
	device.AddChannel(&c.Channel)
	addInstallTest(&c.Channel)

	// add ACTUAL_TEMPERATURE parameter
	c.temperature = NewFloatParameter("ACTUAL_TEMPERATURE")
	c.temperature.description.Max = 3276.7
	c.temperature.description.Min = -3276.8
	c.temperature.description.Unit = "°C"
	c.temperature.OnSetValue = func(value float64) bool {
		if c.OnSetTemperature != nil {
			return c.OnSetTemperature(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.temperature)

	// add ACTUAL_TEMPERATURE_STATUS parameter
	c.temperatureStatus = NewIntParameter("ACTUAL_TEMPERATURE_STATUS")
	c.temperatureStatus.description.Type = itf.ParameterTypeEnum
	c.temperatureStatus.description.Default = "NORMAL"
	c.temperatureStatus.description.Max = "UNDERFLOW"
	c.temperatureStatus.description.Min = "NORMAL"
	c.temperatureStatus.description.ValueList = []string{"NORMAL", "UNKNOWN", "OVERFLOW", "UNDERFLOW"}
	c.temperatureStatus.OnSetValue = func(value int) bool {
		if c.OnSetTemperatureStatus != nil {
			return c.OnSetTemperatureStatus(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.temperatureStatus)

	// add HUMIDITY parameter
	c.humidity = NewIntParameter("HUMIDITY")
	c.humidity.description.Max = 100
	c.humidity.description.Min = 0
	c.humidity.description.Unit = "%"
	c.humidity.OnSetValue = func(value int) bool {
		if c.OnSetHumidity != nil {
			return c.OnSetHumidity(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.humidity)

	// add HUMIDITY_STATUS parameter
	c.humidityStatus = NewIntParameter("HUMIDITY_STATUS")
	c.humidityStatus.description.Type = itf.ParameterTypeEnum
	c.humidityStatus.description.Default = "NORMAL"
	c.humidityStatus.description.Max = "UNDERFLOW"
	c.humidityStatus.description.Min = "NORMAL"
	c.humidityStatus.description.ValueList = []string{"NORMAL", "UNKNOWN", "OVERFLOW", "UNDERFLOW"}
	c.humidityStatus.OnSetValue = func(value int) bool {
		if c.OnSetHumidityStatus != nil {
			return c.OnSetHumidityStatus(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.humidityStatus)

	return c
}

// SetTemperature sets the temperature of the sensor.
func (c *TemperatureChannel) SetTemperature(value float64) {
	c.temperature.InternalSetValue(value)
}

// Temperature returns the temperature of the sensor.
func (c *TemperatureChannel) Temperature() float64 {
	return c.temperature.Value().(float64)
}

// SetTemperatureStatus sets the temperature status of the sensor.
func (c *TemperatureChannel) SetTemperatureStatus(value int) {
	c.temperatureStatus.InternalSetValue(value)
}

// TemperatureStatus returns the temperature status of the sensor.
func (c *TemperatureChannel) TemperatureStatus() int {
	return c.temperatureStatus.Value().(int)
}

// SetHumidity sets the humidity of the sensor.
func (c *TemperatureChannel) SetHumidity(value int) {
	c.humidity.InternalSetValue(value)
}

// Humidity returns the humidity of the sensor.
func (c *TemperatureChannel) Humidity() int {
	return c.humidity.Value().(int)
}

// SetHumidityStatus sets the temperature status of the sensor.
func (c *TemperatureChannel) SetHumidityStatus(value int) {
	c.humidityStatus.InternalSetValue(value)
}

// HumidityStatus returns the humidity status of the sensor.
func (c *TemperatureChannel) HumidityStatus() int {
	return c.humidityStatus.Value().(int)
}

// PowerMeterChannel implements a HM power meter channel (e.g. HM-ES-PMSw1-Pl:1).
type PowerMeterChannel struct {
	Channel

	// These callbacks are executed when an external system wants to change the
	// values. Only if the function returns true, the value is actually set.
	OnSetEnergyCounter func(value float64) (ok bool)
	OnSetPower         func(value float64) (ok bool)
	OnSetCurrent       func(value float64) (ok bool)
	OnSetVoltage       func(value float64) (ok bool)
	OnSetFrequency     func(value float64) (ok bool)

	energyCounter *FloatParameter
	power         *FloatParameter
	current       *FloatParameter
	voltage       *FloatParameter
	frequency     *FloatParameter
}

// NewPowerMeterChannel creates a new HM power meter channel and adds it to the
// device.
func NewPowerMeterChannel(device *Device) *PowerMeterChannel {
	c := new(PowerMeterChannel)
	c.Channel.Init("POWERMETER")
	// adding channel to device also initializes some fields
	device.AddChannel(&c.Channel)
	addInstallTest(&c.Channel)

	// add ENERGY_COUNTER parameter
	c.energyCounter = NewFloatParameter("ENERGY_COUNTER")
	c.energyCounter.description.Max = 838860.7
	c.energyCounter.description.Min = 0.0
	c.energyCounter.description.Unit = "Wh"
	c.energyCounter.description.Control = "POWERMETER.ENERGY_COUNTER"
	c.energyCounter.description.TabOrder = 0
	c.energyCounter.OnSetValue = func(value float64) bool {
		if c.OnSetEnergyCounter != nil {
			return c.OnSetEnergyCounter(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.energyCounter)

	// add POWER parameter
	c.power = NewFloatParameter("POWER")
	c.power.description.Max = 167772.15
	c.power.description.Min = 0.0
	c.power.description.Unit = "W"
	c.power.description.Control = "POWERMETER.POWER"
	c.power.description.TabOrder = 1
	c.power.OnSetValue = func(value float64) bool {
		if c.OnSetPower != nil {
			return c.OnSetPower(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.power)

	// add CURRENT parameter
	c.current = NewFloatParameter("CURRENT")
	c.current.description.Max = 65535.0
	c.current.description.Min = 0.0
	c.current.description.Unit = "mA"
	c.current.description.Control = "POWERMETER.CURRENT"
	c.current.description.TabOrder = 2
	c.current.OnSetValue = func(value float64) bool {
		if c.OnSetCurrent != nil {
			return c.OnSetCurrent(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.current)

	// add VOLTAGE parameter
	c.voltage = NewFloatParameter("VOLTAGE")
	c.voltage.description.Max = 6553.5
	c.voltage.description.Min = 0.0
	c.voltage.description.Unit = "V"
	c.voltage.description.Control = "POWERMETER.VOLTAGE"
	c.voltage.description.TabOrder = 3
	c.voltage.OnSetValue = func(value float64) bool {
		if c.OnSetVoltage != nil {
			return c.OnSetVoltage(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.voltage)

	// add FREQUENCY parameter
	c.frequency = NewFloatParameter("FREQUENCY")
	c.frequency.description.Max = 51.27
	c.frequency.description.Min = 48.72
	c.frequency.description.Unit = "Hz"
	c.frequency.description.Control = "POWERMETER.FREQUENCY"
	c.frequency.description.TabOrder = 4
	c.frequency.OnSetValue = func(value float64) bool {
		if c.OnSetFrequency != nil {
			return c.OnSetFrequency(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.frequency)

	// Add bool parameter with the fixed value true. This is needed so that
	// meter overflows are better handled by the CCU total energy meter.
	boot := NewBoolParameter("BOOT")
	boot.description.Control = "POWERMETER.BOOT"
	// not writeable
	boot.description.Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	// internal
	boot.description.Flags = itf.ParameterFlagVisible | itf.ParameterFlagInternal
	boot.description.TabOrder = 5
	// fixed value true
	boot.InternalSetValue(true)
	boot.OnSetValue = func(value bool) bool {
		return false
	}
	c.AddValueParam(boot)

	return c
}

func (c *PowerMeterChannel) SetEnergyCounter(value float64) {
	c.energyCounter.InternalSetValue(value)
}

func (c *PowerMeterChannel) EnergyCounter() float64 {
	return c.energyCounter.Value().(float64)
}

func (c *PowerMeterChannel) SetPower(value float64) {
	c.power.InternalSetValue(value)
}

func (c *PowerMeterChannel) Power() float64 {
	return c.power.Value().(float64)
}

func (c *PowerMeterChannel) SetCurrent(value float64) {
	c.current.InternalSetValue(value)
}

func (c *PowerMeterChannel) Current() float64 {
	return c.current.Value().(float64)
}

func (c *PowerMeterChannel) SetVoltage(value float64) {
	c.voltage.InternalSetValue(value)
}

func (c *PowerMeterChannel) Voltage() float64 {
	return c.voltage.Value().(float64)
}

func (c *PowerMeterChannel) SetFrequency(value float64) {
	c.frequency.InternalSetValue(value)
}

func (c *PowerMeterChannel) Frequency() float64 {
	return c.frequency.Value().(float64)
}

// EnergyCounterChannel implements a HM energy meter channel (e.g.
// HM-ES-TX-WM:1) of type POWERMETER_IEC1.
type EnergyCounterChannel struct {
	Channel

	// These callbacks are executed when an external system wants to change the
	// values. Only if the function returns true, the value is actually set.
	OnSetEnergyCounter func(value float64) (ok bool)
	OnSetPower         func(value float64) (ok bool)

	energyCounter *FloatParameter
	power         *FloatParameter
}

// NewEnergyCounterChannel creates a new HM energy meter channel and adds it to
// the device.
func NewEnergyCounterChannel(device *Device) *EnergyCounterChannel {
	c := new(EnergyCounterChannel)
	c.Channel.Init("POWERMETER_IEC1")
	// adding channel to device also initializes some fields
	device.AddChannel(&c.Channel)
	addInstallTest(&c.Channel)

	// add ENERGY_COUNTER parameter
	c.energyCounter = NewFloatParameter("IEC_ENERGY_COUNTER")
	//  The associated CCU energy meter, an automatically created script, uses
	//  Max to calculate overruns.
	c.energyCounter.description.Max = 1000000.0
	c.energyCounter.description.Min = 0.0
	c.energyCounter.description.Unit = "kWh"
	c.energyCounter.description.Control = "POWERMETER_IEC1.IEC_ENERGY_COUNTER"
	c.energyCounter.description.TabOrder = 0
	c.energyCounter.OnSetValue = func(value float64) bool {
		if c.OnSetEnergyCounter != nil {
			return c.OnSetEnergyCounter(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.energyCounter)

	// add POWER parameter
	c.power = NewFloatParameter("IEC_POWER")
	c.power.description.Unit = "W"
	c.power.description.Control = "POWERMETER_IEC1.IEC_POWER"
	c.power.description.TabOrder = 1
	c.power.OnSetValue = func(value float64) bool {
		if c.OnSetPower != nil {
			return c.OnSetPower(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.power)
	return c
}

func (c *EnergyCounterChannel) SetEnergyCounter(value float64) {
	c.energyCounter.InternalSetValue(value)
}

func (c *EnergyCounterChannel) EnergyCounter() float64 {
	return c.energyCounter.Value().(float64)
}

func (c *EnergyCounterChannel) SetPower(value float64) {
	c.power.InternalSetValue(value)
}

func (c *EnergyCounterChannel) Power() float64 {
	return c.power.Value().(float64)
}

// GasCounterChannel implements a HM gas meter channel (e.g. HM-ES-TX-WM:1) of
// type POWERMETER_IEC1.
type GasCounterChannel struct {
	Channel

	// These callbacks are executed when an external system wants to change the
	// values. Only if the function returns true, the value is actually set.
	OnSetEnergyCounter func(value float64) (ok bool)
	OnSetPower         func(value float64) (ok bool)

	energyCounter *FloatParameter
	power         *FloatParameter
}

// NewGasCounterChannel creates a new HM gas meter channel and adds it to the
// device.
func NewGasCounterChannel(device *Device) *GasCounterChannel {
	c := new(GasCounterChannel)
	c.Channel.Init("POWERMETER_IEC1")
	// adding channel to device also initializes some fields
	device.AddChannel(&c.Channel)
	addInstallTest(&c.Channel)

	// add GAS_ENERGY_COUNTER parameter
	c.energyCounter = NewFloatParameter("GAS_ENERGY_COUNTER")
	//  The associated CCU energy meter, an automatically created script, uses
	//  Max to calculate overruns.
	c.energyCounter.Description().Max = 1000000.0
	c.energyCounter.Description().Min = 0.0
	c.energyCounter.Description().Unit = "m3"
	c.energyCounter.Description().Control = "POWERMETER_IEC1.GAS_ENERGY_COUNTER"
	c.energyCounter.Description().TabOrder = 0
	c.energyCounter.OnSetValue = func(value float64) bool {
		if c.OnSetEnergyCounter != nil {
			return c.OnSetEnergyCounter(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.energyCounter)

	// add GAS_POWER parameter
	c.power = NewFloatParameter("GAS_POWER")
	c.power.Description().Unit = "m3/h"
	c.power.Description().Control = "POWERMETER_IEC1.GAS_POWER"
	c.power.Description().TabOrder = 1
	c.power.OnSetValue = func(value float64) bool {
		if c.OnSetPower != nil {
			return c.OnSetPower(value)
		} else {
			return true
		}
	}
	c.AddValueParam(c.power)

	// The following parameters are only required for a correct view of the
	// device in the web UI and the CCU scripts for the counter overflows.

	// add MASTER parameter METER_TYPE with fixed value 0
	meterType := NewIntParameter("METER_TYPE")
	meterType.Description().Type = itf.ParameterTypeEnum
	meterType.Description().ValueList = []string{"GAS-SENSOR", "IR-SENSOR", "LED-SENSOR", "IEC-SENSOR", "UNKOWN"}
	meterType.Description().Min = 0
	meterType.Description().Max = len(meterType.Description().ValueList) - 1
	meterType.Description().Default = 0
	// not writeable
	meterType.Description().Operations = itf.ParameterOperationRead
	// not visible, internal
	meterType.Description().Flags = itf.ParameterFlagInternal
	// fixed value "GAS-SENSOR"
	meterType.value.Store(0)
	c.AddMasterParam(meterType)

	// add ENERGY_COUNTER parameter
	fakeEnergyCounter := NewFloatParameter("ENERGY_COUNTER")
	fakeEnergyCounter.Description().Max = 1000000.0
	fakeEnergyCounter.Description().Min = 0.0
	fakeEnergyCounter.Description().Unit = "Wh"
	fakeEnergyCounter.Description().Control = "POWERMETER_IEC1.ENERGY_COUNTER"
	fakeEnergyCounter.Description().TabOrder = 2
	// not writeable
	fakeEnergyCounter.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.AddValueParam(fakeEnergyCounter)

	// add POWER parameter
	fakePower := NewFloatParameter("POWER")
	fakePower.Description().Unit = "W"
	fakePower.Description().Control = "POWERMETER_IEC1.POWER"
	fakePower.Description().TabOrder = 3
	// not writeable
	fakePower.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.AddValueParam(fakePower)

	// add IEC_ENERGY_COUNTER parameter
	fakeIECEnergyCounter := NewFloatParameter("IEC_ENERGY_COUNTER")
	fakeIECEnergyCounter.Description().Max = 1000000.0
	fakeIECEnergyCounter.Description().Min = 0.0
	fakeIECEnergyCounter.Description().Unit = "kWh"
	fakeIECEnergyCounter.Description().Control = "POWERMETER_IEC1.IEC_ENERGY_COUNTER"
	fakeIECEnergyCounter.Description().TabOrder = 4
	// not writeable
	fakeIECEnergyCounter.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.AddValueParam(fakeIECEnergyCounter)

	// add IEC_POWER parameter
	fakeIECPower := NewFloatParameter("IEC_POWER")
	fakeIECPower.Description().Unit = "W"
	fakeIECPower.Description().Control = "POWERMETER_IEC1.IEC_POWER"
	fakeIECPower.Description().TabOrder = 5
	// not writeable
	fakeIECPower.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.AddValueParam(fakeIECPower)

	// add BOOT parameter with the fixed value false
	fakeBoot := NewBoolParameter("BOOT")
	fakeBoot.Description().Control = "POWERMETER_IEC1.BOOT"
	fakeBoot.Description().TabOrder = 6
	// not writeable
	fakeBoot.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	// internal
	fakeBoot.Description().Flags = itf.ParameterFlagVisible | itf.ParameterFlagInternal
	// fixed value false
	fakeBoot.value.Store(false)
	c.AddValueParam(fakeBoot)

	return c
}

func (c *GasCounterChannel) SetEnergyCounter(value float64) {
	c.energyCounter.InternalSetValue(value)
}

func (c *GasCounterChannel) EnergyCounter() float64 {
	return c.energyCounter.Value().(float64)
}

func (c *GasCounterChannel) SetPower(value float64) {
	c.power.InternalSetValue(value)
}

func (c *GasCounterChannel) Power() float64 {
	return c.power.Value().(float64)
}
