package tcp

import (
	"context"
	"crypto/rsa"
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
	listener    *net.TCPListener
	banList     map[addr.IP]bool
	banMu       *sync.RWMutex
	closed      bool
	closedMu    *sync.RWMutex
	dialTimeout time.Duration
	privateKey  *rsa.PrivateKey
}

func New(tcpAddr *net.TCPAddr) (*Node, error) {
	listener, err := listenConfig.Listen(context.Background(), tcpAddr.Network(), tcpAddr.String())
	if err != nil {
		return nil, err
	}
	return &Node{
		listener:    listener.(*net.TCPListener),
		banList:     map[addr.IP]bool{},
		banMu:       &sync.RWMutex{},
		closed:      false,
		closedMu:    &sync.RWMutex{},
		dialTimeout: dialTimeout,
	}, nil
}

func (node *Node) SetPrivateKey(key *rsa.PrivateKey) {
	node.privateKey = key
}

func (node *Node) Dial(raddr addr.Addr) (conn.Conn, error) {
	node.closedMu.RLock()
	if node.closed {
		node.closedMu.RUnlock()
		return nil, NodeClosedErr
	}
	node.closedMu.RUnlock()
	dialer := net.Dialer{
		Timeout:   node.dialTimeout,
		Deadline:  time.Now().Add(node.dialTimeout),
		Control:   control,
		LocalAddr: node.listener.Addr(),
	}
	tcpAddr := raddr.ToTCPAddr()
	tcpConn, err := dialer.Dial(tcpAddr.Network(), tcpAddr.String())
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
		if !banned {
			return &Conn{conn: tcpConn}, nil
		}
	}
}

func (node *Node) Ban(ip addr.IP) {
	node.banMu.Lock()
	defer node.banMu.Unlock()
	node.banList[ip] = true
}

func (node *Node) RevokeBan(ip addr.IP) {
	node.banMu.Lock()
	defer node.banMu.Unlock()
	delete(node.banList, ip)
}

func (node *Node) SetDeadline(t time.Time) error {
	return node.listener.SetDeadline(t)
}

func (node *Node) SetDialTimeout(d time.Duration) {
	node.dialTimeout = d
}

func (node *Node) Close() error {
	node.closedMu.Lock()
	defer node.closedMu.Unlock()
	node.closed = true
	return node.listener.Close()
}

func (node *Node) Addr() addr.Addr {
	return addr.TCPToAddr(node.listener.Addr().(*net.TCPAddr))
}
