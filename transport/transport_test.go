package transport_test

import (
	"github.com/cevatbarisyilmaz/go-imp/addr"
	"github.com/cevatbarisyilmaz/go-imp/transport"
	"net"
	"testing"
)

func Test(t *testing.T) {
	_, err := transport.New(addr.NewAddr(addr.NetIPToIP(net.IPv4(127, 0, 0, 2)), addr.IntToPort(1789), addr.ProtoTCP))
	if err != nil {
		t.Fatal(err)
	}
}
