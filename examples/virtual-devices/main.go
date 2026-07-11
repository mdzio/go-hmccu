/*
This is an example of providing virtual devices to the CCU via an interface
process.

Command line arguments:

	-ccu address
		address of the CCU (default "127.0.0.1")
	-devices number
		number of devices (default 2)
	-channels number
		number of channels of each channel type (default 2)
	-http port
		port for serving HTTP (default 2124)
	-log severity
		specifies the minimum severity of printed log messages: ERROR, WARN, INFO, DEBUG
	-prefix prefix
		prefix for device serial numbers (default "VDT")

Guide:

To register the interface at the logic layer of the CCU, the CCU file
/etc/config/InterfacesList.xml must be modified. Add a new ipc XML element below
the root element. Replace the IP address 192.168.0.20 with the host address
running this example:

	<ipc>
	    <name>My-Virtual-Devices</name>
	    <url>xmlrpc://192.168.0.20:2124/RPC2</url>
	    <info>My Virtual Devices</info>
	</ipc>

Authentication of the CCU API under Control Panel -> Security must be switched
off.

If not running on the CCU, specify the address of the CCU on the command line.
Replace the IP address 192.168.0.10 with the address of the CCU:

	-ccu 192.168.0.10

Then restart the ReGaHss and HMServer with these commands:

	/etc/init.d/S70ReGaHss restart
	/etc/init.d/S62HMServer restart

On RaspberryMatic, monit may possibly interfere. It should therefore be
deactivated:

	monit unmonitor all

Do not forget to restore the InterfacesList.xml afterwards and to restart the
ReGaHss and HMServer. The file is automatically restored when rebooting the CCU.
*/
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/mdzio/go-hmccu/v2/itf"
	"github.com/mdzio/go-hmccu/v2/itf/vdevices"
	"github.com/mdzio/go-hmccu/v2/itf/xmlrpc"
)

const (
	// default CCU RPC path
	rpcPath = "/RPC2"
)

// wrap slog.Level to implement flag.Value interface
type logLevel struct {
	slog.Level
}

// Set implements flag.Value interface
func (l *logLevel) Set(s string) error {
	return l.UnmarshalText([]byte(s))
}

var (
	log = slog.With("component", "main")

	httpPort    = flag.Int("http", 2124, "`port` for serving HTTP")
	ccuAddress  = flag.String("ccu", "127.0.0.1", "`address` of the CCU")
	numDevices  = flag.Int("devices", 2, "`number` of devices")
	numChannels = flag.Int("channels", 2, "`number` of channels of each channel type")
	prefix      = flag.String("prefix", "VDT", "`prefix` for device serial numbers")
	logLvl      logLevel // default slog.LevelInfo
)

func init() {
	flag.Var(
		&logLvl,
		"log",
		"specifies the minimum `severity` of printed log messages: ERROR, WARN, INFO, DEBUG",
	)
}

func run() error {
	// parse command line
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage of virtual-devices:")
		flag.PrintDefaults()
	}
	// flag.Parse calls os.Exit(2) on error
	flag.Parse()

	// set log level
	slog.SetLogLoggerLevel(logLvl.Level)

	// virtual device container
	vdevs := vdevices.NewContainer()

	// virtual devices handler
	vdevHandler := vdevices.NewHandler(*ccuAddress, false, vdevs, func(string) {})
	defer vdevHandler.Close()
	vdevs.Synchronizer = vdevHandler

	// create devices
	for devidx := 0; devidx < *numDevices; devidx++ {
		addr := fmt.Sprintf("%s%03d", *prefix, devidx)
		dev := vdevices.NewDevice(addr, "HmIP-MIO16-PCB", vdevHandler)
		dev.OnDispose = func() {
			log.Debug("Device disposed", "address", dev.Description().Address)
		}
		bp := vdevices.NewBoolParameter("BOOL_PARAM")
		dev.AddMasterParam(bp)

		// maintenance channel
		mch := vdevices.NewMaintenanceChannel(dev)
		mch.OnDispose = func() {
			log.Debug("Channel disposed", "address", mch.Description().Address)
		}
		log.Info("Created maintenance channel", "address", mch.Description().Address)

		// switch channels
		for chidx := 0; chidx < *numChannels; chidx++ {
			sch := vdevices.NewSwitchChannel(dev)
			sch.OnSetState = func(value bool) bool {
				log.Debug("Switch channel is set", "address", sch.Description().Address, "value", value)
				return true
			}
			sch.OnDispose = func() {
				log.Debug("Channel disposed", "address", sch.Description().Address)
			}
			bp = vdevices.NewBoolParameter("BOOL_PARAM")
			sch.AddMasterParam(bp)
			log.Info("Created switch channel", "address", sch.Description().Address)
		}

		// key channels
		for chidx := 0; chidx < *numChannels; chidx++ {
			kch := vdevices.NewKeyChannel(dev)
			kch.OnPressShort = func() bool {
				log.Debug("Key channel pressed short", "address", kch.Description().Address)
				return true
			}
			kch.OnPressLong = func() bool {
				log.Debug("Key channel pressed long", "address", kch.Description().Address)
				return true
			}
			log.Info("Created key channel", "address", kch.Description().Address)
		}

		vdevs.AddDevice(dev)
		log.Info("Created device", "address", dev.Description().Address)
	}

	// HM RPC dispatcher
	dispatcher := itf.NewDispatcher()
	dispatcher.AddDeviceLayer(vdevHandler)

	// register XML-RPC handler at the HTTP server
	httpHandler := &xmlrpc.Handler{Dispatcher: dispatcher}
	http.Handle(rpcPath, httpHandler)

	// run HTTP server
	log.Info("Starting HTTP server", "port", *httpPort)
	return http.ListenAndServe(":"+strconv.Itoa(*httpPort), nil)
}

func main() {
	err := run()
	// log fatal error
	if err != nil {
		log.Error("Application failed", "error", err)
		os.Exit(1)
	}
	os.Exit(0)
}
