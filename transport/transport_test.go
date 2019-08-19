package transport_test

import (
	"github.com/cevatbarisyilmaz/go-imp/addr"
	"github.com/cevatbarisyilmaz/go-imp/transport"
	conn "github.com/cevatbarisyilmaz/go-imp/transport/conn"
	"sync"
	"testing"
)

func Test(t *testing.T) {
	for proto := addr.ProtoTCP; true; proto = addr.ProtoUDP {
		dialer, err := transport.New(addr.NewAddr(addr.IPv4(127, 0, 0, 1), addr.IntToPort(0), proto))
		if err != nil {
			t.Fatal(err)
		}
		listener, err := transport.New(addr.NewAddr(addr.IPv4(127, 0, 0, 1), addr.IntToPort(0), proto))
		if err != nil {
			t.Fatal(err)
		}
		testNode(t, dialer, listener)
		err = dialer.Close()
		if err != nil {
			t.Error(err)
		}
		err = listener.Close()
		if err != nil {
			t.Error(err)
		}
		if proto == addr.ProtoUDP {
			return
		}
	}
}

func testNode(t *testing.T, dialer, listener transport.Node) {
	var c1, c2 conn.Conn
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		var err error
		c1, err = dialer.Dial(listener.Addr())
		if err != nil {
			t.Fatal(err)
		}
		err = c1.Write([]byte{0})
		if err != nil {
			t.Fatal(err)
		}
		wg.Done()
	}()
	go func() {
		var err error
		c2, err = listener.Accept()
		if err != nil {
			t.Fatal(err)
		}
		msg, err := c2.Read()
		if err != nil {
			t.Fatal(err)
		}
		if len(msg) != 1 || msg[0] != 0 {
			t.Fatal("received wrong message")
		}
		wg.Done()
	}()
	wg.Wait()
	conn.TestConn(t, c1, c2)
}
