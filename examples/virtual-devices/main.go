/*
This is an example of providing virtual devices to the CCU via an interface
process.

To register the interface at the logic layer of the CCU, the CCU file
/etc/config/InterfacesList.xml must be modified. Add a new ipc XML element below
the root element. Replace the IP address 192.168.0.20 with the host address
running this example:

    <ipc>
        <name>My-Virtual-Devices</name>
        <url>xmlrpc://192.168.0.20:2124/RPC2</url>
        <info>My Virtual Devices</info>
    </ipc>

Authentication of the CCU API under Control Panel → Security must be switched
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
	"net/http"
	"os"
	"strconv"

	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"github.com/mdzio/go-logging"
)

const (
	// default CCU RPC path
	rpcPath = "/RPC2"
)

var (
	log = logging.Get("main")

	logLevel    = logging.InfoLevel
	httpPort    = flag.Int("http", 2124, "`port` for serving HTTP")
	ccuAddress  = flag.String("ccu", "127.0.0.1", "`address` of the CCU")
	numDevices  = flag.Int("devices", 2, "`number` of devices")
	numChannels = flag.Int("channels", 2, "`number` of channels of each channel type")
	prefix      = flag.String("prefix", "VDT", "`prefix` for device serial numbers")
)

func init() {
	flag.Var(
		&logLevel,
		"log",
		"specifies the minimum `severity` of printed log messages: off, error, warning, info, debug or trace",
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
	// set log options
	logging.SetLevel(logLevel)

	// virtual device container
	vdevs := vdevices.NewContainer()

	// virtual devices handler
	vdevHandler := vdevices.NewHandler(*ccuAddress, vdevs, func(string) {})
	defer vdevHandler.Close()
	vdevs.Synchronizer = vdevHandler

	// create devices
	for devidx := 0; devidx < *numDevices; devidx++ {
		addr := fmt.Sprintf("%s%03d", *prefix, devidx)
		dev := vdevices.NewDevice(addr, "HmIP-MIO16-PCB", vdevHandler)
		dev.OnDispose = func() {
			log.Debugf("Device %s is disposed", dev.Description().Address)
		}
		bp := vdevices.NewBoolParameter("BOOL_PARAM")
		dev.AddMasterParam(&bp.Parameter)

		// maintenance channel
		mch := vdevices.NewMaintenanceChannel(dev)
		mch.OnDispose = func() {
			log.Debugf("Channel %s is disposed", mch.Description().Address)
		}
		log.Infof("Created maintenance channel: %s", mch.Description().Address)

		// switch channels
		for chidx := 0; chidx < *numChannels; chidx++ {
			sch := vdevices.NewSwitchChannel(dev)
			sch.OnSetState = func(value bool) bool {
				log.Debugf("Switch channel %s is set: %t", sch.Description().Address, value)
				return true
			}
			sch.OnDispose = func() {
				log.Debugf("Channel %s is disposed", sch.Description().Address)
			}
			bp = vdevices.NewBoolParameter("BOOL_PARAM")
			sch.AddMasterParam(&bp.Parameter)
			log.Infof("Created switch channel: %s", sch.Description().Address)
		}

		// key channels
		for chidx := 0; chidx < *numChannels; chidx++ {
			kch := vdevices.NewKeyChannel(dev)
			kch.OnPressShort = func() bool {
				log.Debugf("Key channel %s is pressed short", kch.Description().Address)
				return true
			}
			kch.OnPressLong = func() bool {
				log.Debugf("Key channel %s is pressed long", kch.Description().Address)
				return true
			}
			log.Infof("Created key channel: %s", kch.Description().Address)
		}

		vdevs.AddDevice(dev)
		log.Infof("Created device: %s", dev.Description().Address)
	}

	// HM RPC dispatcher
	dispatcher := itf.NewDispatcher()
	dispatcher.AddDeviceLayer(vdevHandler)

	// register XML-RPC handler at the HTTP server
	httpHandler := &xmlrpc.Handler{Dispatcher: dispatcher}
	http.Handle(rpcPath, httpHandler)

	// run HTTP server
	log.Infof("Starting HTTP server on port %d", *httpPort)
	return http.ListenAndServe(":"+strconv.Itoa(*httpPort), nil)
}

func main() {
	err := run()
	// log fatal error
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	os.Exit(0)
}
