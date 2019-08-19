package tcp

import (
	"context"
	"errors"
	"github.com/cevatbarisyilmaz/go-imp/addr"
	conn "github.com/cevatbarisyilmaz/go-imp/transport/conn"
	"net"
	"sync"
	"time"
)

const dialTimeout = time.Minute

var NodeClosedErr = errors.New("this IMP node is closed")

var listenConfig = net.ListenConfig{
	Control: control,
}

type Node struct {
	listener *net.TCPListener
	banList  map[addr.IP]bool
	banMu    *sync.RWMutex
	closed   bool
	closedMu *sync.RWMutex
}

func New(laddr addr.Addr) (*Node, error) {
	listener, err := listenConfig.Listen(context.Background(), laddr.Network(), laddr.String())
	if err != nil {
		return nil, err
	}
	return &Node{
		listener: listener.(*net.TCPListener),
		banList:  map[addr.IP]bool{},
		banMu:    &sync.RWMutex{},
		closed:   false,
		closedMu: &sync.RWMutex{},
	}, nil
}

func (node *Node) Dial(raddr addr.Addr) (conn.Conn, error) {
	node.closedMu.RLock()
	if node.closed {
		node.closedMu.RUnlock()
		return nil, NodeClosedErr
	}
	node.closedMu.RUnlock()
	dialer := net.Dialer{
		Timeout:   dialTimeout,
		Deadline:  time.Now().Add(dialTimeout),
		Control:   control,
		LocalAddr: node.listener.Addr(),
	}
	tcpConn, err := dialer.Dial(raddr.Network(), raddr.String())
	if err != nil {
		return nil, err
	}
	return &Conn{conn: tcpConn.(*net.TCPConn)}, nil
}

func (node *Node) Accept() (conn.Conn, error) {
	for {
		tcpConn, err := node.listener.AcceptTCP()
		if err != nil {
			return nil, err
		}
		node.banMu.RLock()
		banned := node.banList[addr.NetIPToIP(tcpConn.RemoteAddr().(*net.TCPAddr).IP)]
		node.banMu.RUnlock()
		if banned {
			_ = tcpConn.Close()
		} else {
			return &Conn{conn: tcpConn}, nil
		}
	}
}

func (node *Node) Ban(ip addr.IP) {
	node.banMu.Lock()
	defer node.banMu.Unlock()
	node.banList[ip] = true
}

func (node *Node) Close() error {
	node.closedMu.Lock()
	node.closed = true
	node.closedMu.Unlock()
	return node.listener.Close()
}

func (node *Node) Addr() addr.Addr {
	return addr.TCPToAddr(node.listener.Addr().(*net.TCPAddr))
}
