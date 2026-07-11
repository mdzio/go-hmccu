/*
This example shows how to read all information about a CCU device. The output is
valid Go code, which can be used for defining virtual devices.

Usage of device-info:

	-ccu address
		address of the CCU (default "127.0.0.1")
	-device address
		address (serial no.) of a CCU device/channel (default "BidCoS-RF:1")
	-list
		list all devices
	-log severity
		specifies the minimum severity of printed log messages: ERROR, WARN, INFO, DEBUG
	-port number
		port number of the CCU interface process: e.g. 2000 (BidCos-Wired), 2001 (BidCos-RF), 2010 (HmIP-RF), 8701 (CUxD), 2121 (CCU-Jack) (default 2001)

Prerequisites:

	Authentication of the XML-RPC interface must be switched off in the CCU's system control.

Build command:

	go build -trimpath "-ldflags=-s -w" .
*/
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/mdzio/go-hmccu/v2/itf"
	"github.com/mdzio/go-hmccu/v2/itf/binrpc"
	"github.com/mdzio/go-hmccu/v2/itf/xmlrpc"
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
	ccu    = flag.String("ccu", "127.0.0.1", "`address` of the CCU")
	port   = flag.Int("port", 2001, "port `number` of the CCU interface process: e.g. 2000 (BidCos-Wired), 2001 (BidCos-RF), 2010 (HmIP-RF), 8701 (CUxD), 2121 (CCU-Jack)")
	device = flag.String("device", "BidCoS-RF:1", "`address` (serial no.) of a CCU device/channel")
	list   = flag.Bool("list", false, "list all devices")
	logLvl logLevel // default slog.LevelInfo
	log    = slog.With("component", "main")
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
		fmt.Fprintln(os.Stderr, "usage of device-info:")
		flag.PrintDefaults()
	}
	// flag.Parse calls os.Exit(2) on error
	flag.Parse()

	// set log level
	slog.SetLogLoggerLevel(logLvl.Level)

	// create interface client
	addr := *ccu + ":" + strconv.Itoa(*port)
	if *port == 2121 {
		addr += "/RPC3"
	}
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
		log.Error("Application failed", "error", err)
		os.Exit(1)
	}
	os.Exit(0)
}
