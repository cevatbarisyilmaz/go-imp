package tcp_test

import (
	"github.com/cevatbarisyilmaz/go-imp/addr"
	"github.com/cevatbarisyilmaz/go-imp/transport/tcp"
	"net"
	"sync"
	"testing"
)

func Test(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		node, err := tcp.New(&net.TCPAddr{
			IP:   net.IPv4(127, 0, 0, 2),
			Port: 1789,
		})
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
		_, err = conn.Write([]byte("Hey!"))
		if err != nil {
			t.Fatal(err)
		}
		wg.Done()
	}()
	go func() {
		node, err := tcp.New(&net.TCPAddr{
			IP:   net.IPv4(127, 0, 0, 2),
			Port: 1919,
		})
		if err != nil {
			t.Fatal(err)
		}
		conn, err := node.Dial(addr.NewAddr(addr.NetIPToIP(net.IPv4(127, 0, 0, 2)), addr.IntToPort(1789), addr.ProtoTCP))
		if err != nil {
			t.Fatal(err)
		}
		_, err = conn.Write([]byte("Hi!"))
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
	}()
	wg.Wait()
}
