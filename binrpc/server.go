package binrpc

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
)

type iHandler interface {
	ServeTCP(conn net.Conn)
}

func Server(addr string, handler iHandler) {
	listenAddr := strings.Replace(addr, "xmlrpc_bin://", "", 1)
	l, err := net.Listen("tcp4", listenAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()
	rand.Seed(time.Now().Unix())

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go handler.ServeTCP(c)
	}
}
