package imp_test

import (
	"github.com/cevatbarisyilmaz/go-imp"
	"github.com/cevatbarisyilmaz/go-imp/addr"
	"net"
	"sync"
	"testing"
)

func Test(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		node := imp.New()
		err := node.AddAddr(addr.NewAddr(addr.NetIPToIP(net.IPv4(127, 0, 0, 2)), addr.IntToPort(1789), addr.ProtoTCP))
		if err != nil {
			t.Fatal(err)
		}
		conn, err := node.Accept()
		if err != nil {
			t.Fatal(err)
		}
		msg, err := conn.Read()
		if err != nil {
			t.Fatal(err)
		}
		if string(msg) != "Hi!" {
			t.Fatal("wrong message recieved: ", string(msg))
		}
		err = conn.Write([]byte("Hey!"))
		if err != nil {
			t.Fatal(err)
		}
		wg.Done()
	}()
	node := imp.New()
	err := node.AddAddr(addr.NewAddr(addr.NetIPToIP(net.IPv4(127, 0, 0, 2)), addr.IntToPort(1919), addr.ProtoTCP))
	if err != nil {
		t.Fatal(err)
	}
	conn, err := node.Dial(addr.NewAddr(addr.NetIPToIP(net.IPv4(127, 0, 0, 2)), addr.IntToPort(1789), addr.ProtoTCP))
	if err != nil {
		t.Fatal(err)
	}
	err = conn.Write([]byte("Hi!"))
	if err != nil {
		t.Fatal(err)
	}
	msg, err := conn.Read()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != "Hey!" {
		t.Fatal("wrong message recieved: ", string(msg))
	}
	wg.Done()
	wg.Wait()
}
