# go-hmccu

[go-hmccu](https://github.com/mdzio/go-hmccu) provides functionality in the programming language [Go](https://golang.org) to communicate with the CCU of the [HomeMatic home automation system](https://www.eq-3.de/produkte/homematic.html). Data points can be explored, read and written.

## Example programs

### device-info

This programme can be used to read out the meta information on devices and channels from a CCU interface process.

In order for the interface processes to be accessible via the network, the firewall must be set accordingly on the CCU and the authentication for the interface processes must be deactivated.

```
usage of device-info:
  -ccu address
        address of the CCU (default "127.0.0.1")
  -device address
        address (serial no.) of a CCU device/channel (default "BidCoS-RF:1")
  -list
        list all devices
  -log severity
        specifies the minimum severity of printed log messages: off, error, warning, info, debug or trace (default INFO)
  -port number
        port number of the CCU interface process: e.g. 2000 (BidCos-Wired), 2001 (BidCos-RF), 2010 (HmIP-RF), 8701 (CUxD), 2121 (CCU-Jack) (default 2001)
```

## Authors

* [Mathias Dz.](https://github.com/mdzio)
* [twendt](https://github.com/twendt) (BIN-RPC for CUxD)

## License

This work is licensed under the [GNU General Public License V3](LICENSE.txt).

## Credits

* BIN-RPC implementation for NodeJS by Sebastian Raff (https://github.com/hobbyquaker/binrpc)
