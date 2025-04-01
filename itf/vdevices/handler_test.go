package vdevices

import (
	"bytes"
	"os"
	"testing"

	_ "github.com/mdzio/go-lib/testutil"
)

const expectedInterfaceList = `<?xml version="1.0" encoding="utf-8" ?> 
<interfaces v="1.0">
	<ipc>
	 	<name>BidCos-RF</name>
	 	<url>xmlrpc_bin://127.0.0.1:32001</url> 
	 	<info>BidCos-RF</info> 
	</ipc>
	<ipc>
	 	<name>VirtualDevices</name>
	 	<url>xmlrpc://127.0.0.1:39292/groups</url> 
	 	<info>Virtual Devices</info> 
	</ipc>
	<ipc>
	 	<name>HmIP-RF</name>
	 	<url>xmlrpc://127.0.0.1:32010</url>
	 	<info>HmIP-RF</info>
	</ipc>
	<ipc>
	 	<name>CCU-Jack</name>
	 	<url>xmlrpc://127.0.0.1:2121/RPC3</url>
	 	<info>CCU-Jack</info>
	</ipc>
</interfaces>
`

func TestAddToInterfaceList(t *testing.T) {
	err := AddToInterfaceList(
		"testdata/InterfacesList.xml",
		"out.xml",
		"CCU-Jack",
		"xmlrpc://127.0.0.1:2121/RPC3",
		"CCU-Jack",
	)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("out.xml")

	content, err := os.ReadFile("out.xml")
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != expectedInterfaceList {
		t.Fatalf("unexpected content: %s", string(content))
	}
}

func TestFixStringParam(t *testing.T) {
	cases := []struct {
		in        []byte
		wanted    []byte
		wantedErr bool
	}{
		{[]byte{}, []byte{}, false},
		{[]byte("abc"), []byte("abc"), false},
		{[]byte("ü"), []byte{}, true},
		{[]byte("abcß"), []byte{}, true},
		{[]byte("single quote &#39; double quote &#34;"), []byte(`single quote ' double quote "`), false},
	}
	for _, c := range cases {
		out, err := fixStringParamValue(string(c.in))
		if (err != nil) != c.wantedErr {
			t.Error(c.wantedErr, "!=", err)
		}
		if (err == nil) && !bytes.Equal([]byte(out.(string)), c.wanted) {
			t.Error(string(c.wanted), "!=", out)
		}
	}
}
