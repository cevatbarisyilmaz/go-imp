package udp

import (
	"crypto/rsa"
	"errors"
	"github.com/cevatbarisyilmaz/imp-go/addr"
	"net"
	"sync"
	"time"
)

var TupleErr = errors.New("another IMP connection with the same tuple already exists")
var NodeClosedErr = errors.New("this IMP node is closed")

const maxDatagramSize = 65507
const maxQueueSize = 1024

type timeoutErr struct{}

func (timeoutErr) Error() string {
	return "timeout"
}

func (timeoutErr) Timeout() bool {
	return true
}

func (timeoutErr) Temporary() bool {
	return false
}

type UDPNode struct {
	conn       *net.UDPConn
	conns      map[addr.Addr]*UDPConn
	connsMu    *sync.RWMutex
	queueMap   map[addr.Addr]*UDPConn
	queueChan  chan addr.Addr
	queueMu    *sync.Mutex
	closed     bool
	closedMu   *sync.RWMutex
	deadline   time.Time
	banList    map[addr.IP]bool
	banMu      *sync.RWMutex
	privateKey *rsa.PrivateKey
}

func New(udpAddr *net.UDPAddr) (*UDPNode, error) {
	conn, err := net.ListenUDP(udpAddr.Network(), udpAddr)
	if err != nil {
		return nil, err
	}
	node := &UDPNode{
		conn:      conn,
		conns:     map[addr.Addr]*UDPConn{},
		connsMu:   &sync.RWMutex{},
		queueMap:  map[addr.Addr]*UDPConn{},
		queueChan: make(chan addr.Addr, maxQueueSize),
		queueMu:   &sync.Mutex{},
		closed:    false,
		closedMu:  &sync.RWMutex{},
		deadline:  time.Time{},
	}
	go node.work()
	return node, nil
}

func (node *UDPNode) work() {
	for {
		if node.isClosed() {
			return
		}
		buffer := make([]byte, maxDatagramSize)
		n, udpAddr, err := node.conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}
		go func() {
			buffer = buffer[:n]
			a := addr.UDPToAddr(udpAddr)
			c := node.getConn(a)
			if c != nil {
				c.receive(buffer)
				return
			}
			node.banMu.RLock()
			banned := node.banList[a.IP()]
			node.banMu.RUnlock()
			if banned {
				return
			}
			node.queueMu.Lock()
			defer node.queueMu.Unlock()
			c = node.queueMap[a]
			if c != nil {
				c.receive(buffer)
				return
			}
			if len(node.queueMap) >= maxQueueSize {
				return
			}
			c = &UDPConn{conn: node.conn, raddr: udpAddr, imps: map[imp]bool{}, impsMu: &sync.RWMutex{}}
			node.queueMap[a] = c
			node.queueChan <- a
		}()
	}
}

func (node *UDPNode) isClosed() bool {
	node.closedMu.RLock()
	defer node.closedMu.RUnlock()
	return node.closed
}

func (node *UDPNode) getConn(a addr.Addr) *UDPConn {
	node.connsMu.RLock()
	defer node.connsMu.RUnlock()
	return node.conns[a]
}

func (node *UDPNode) Dial(raddr addr.Addr) (*UDPConn, error) {
	if node.isClosed() {
		return nil, NodeClosedErr
	}
	if node.conns[raddr] != nil {
		return nil, TupleErr
	}
	conn := &UDPConn{conn: node.conn, raddr: raddr.ToUDPAddr(), imps: map[imp]bool{}, impsMu: &sync.RWMutex{}}
	return conn, nil
}

func (node *UDPNode) Accept() (*UDPConn, error) {
	if node.isClosed() {
		return nil, NodeClosedErr
	}
	if node.deadline.Equal(time.Time{}) {
		next := <-node.queueChan
		node.queueMu.Lock()
		defer node.queueMu.Unlock()
		return node.queueMap[next], nil
	}
	select {
	case next := <-node.queueChan:
		node.queueMu.Lock()
		defer node.queueMu.Unlock()
		conn := node.queueMap[next]
		delete(node.queueMap, next)
		return conn, nil
	case <-time.Tick(time.Until(node.deadline)):
		return nil, timeoutErr{}
	}
}

func (node *UDPNode) SetPrivateKey(key *rsa.PrivateKey) {
	node.privateKey = key
}

func (node *UDPNode) Ban(ip addr.IP) {
	node.banMu.Lock()
	defer node.banMu.Unlock()
	node.banList[ip] = true
}

func (node *UDPNode) RevokeBan(ip addr.IP) {
	node.banMu.Lock()
	defer node.banMu.Unlock()
	delete(node.banList, ip)
}

func (node *UDPNode) SetDeadline(t time.Time) error {
	if node.isClosed() {
		return NodeClosedErr
	}
	node.deadline = t
	return nil
}

func (node *UDPNode) Close() error {
	node.closedMu.Lock()
	defer node.closedMu.Unlock()
	node.closed = true
	return node.conn.Close()
}

func (node *UDPNode) Addr() addr.Addr {
	return addr.UDPToAddr(node.conn.LocalAddr().(*net.UDPAddr))
}
