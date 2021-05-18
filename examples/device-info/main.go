/*
This example shows how to read all information about a CCU device. The output is
valid Go code, which can be used for defining virtual devices.
*/
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/binrpc"
	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"github.com/mdzio/go-logging"
)

var (
	log = logging.Get("main")

	logLevel = logging.InfoLevel
	ccu      = flag.String("ccu", "127.0.0.1", "`address` of the CCU")
	port     = flag.Int("port", 2001, "port `number` of the CCU interface process: e.g. 2000 (BidCos-Wired), 2001 (BidCos-RF), 2010 (HmIP-RF), 8701 (CUxD)")
	device   = flag.String("device", "BidCoS-RF:1", "`address` (serial no.) of a CCU device/channel")
	list     = flag.Bool("list", false, "list all devices")
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
		fmt.Fprintln(os.Stderr, "usage of device-info:")
		flag.PrintDefaults()
	}
	// flag.Parse calls os.Exit(2) on error
	flag.Parse()
	// set log options
	logging.SetLevel(logLevel)

	// create interface client
	addr := *ccu + ":" + strconv.Itoa(*port)
	var caller xmlrpc.Caller
	if *port != 8701 {
		caller = &xmlrpc.Client{Addr: addr}
	} else {
		// CUxD
		caller = &binrpc.Client{Addr: addr}
	}
	client := &itf.DeviceLayerClient{Name: addr, Caller: caller}

	// list all devices?
	if *list {
		fmt.Println("=== All devices ===")
		devices, err := client.ListDevices()
		if err != nil {
			return err
		}
		for _, device := range devices {
			fmt.Printf("%#v\n", *device)
		}
		return nil
	}

	// retrieve device description
	fmt.Println("=== Device description:", *device, "===")
	devDescr, err := client.GetDeviceDescription(*device)
	if err != nil {
		return err
	}
	fmt.Printf("%#v\n\n", *devDescr)

	// retrieve paramset descriptions
	for _, ps := range devDescr.Paramsets {
		fmt.Printf("=== Paramset description: %s ===\n", ps)
		psDescr, err := client.GetParamsetDescription(*device, ps)
		if err != nil {
			return err
		}
		for _, v := range psDescr {
			fmt.Printf("%#v\n", *v)
		}
		fmt.Printf("\n")
	}
	return nil
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
