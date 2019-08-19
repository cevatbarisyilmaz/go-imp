package udp

import (
	"errors"
	"github.com/cevatbarisyilmaz/go-imp/addr"
	conn "github.com/cevatbarisyilmaz/go-imp/transport/conn"
	"net"
	"sync"
	"time"
)

var ErrDuplicateTuple = errors.New("another IMP connection with the same tuple already exists")
var ErrNodeClosed = errors.New("this IMP node is closed")

const maxDatagramSize = 65507
const maxQueueSize = 16
const sleepDuration = time.Millisecond
const timeoutDuration = time.Minute
const maxRetry = 32

type Node struct {
	conn      *net.UDPConn
	conns     map[addr.Addr]*Conn
	connsMu   *sync.RWMutex
	queueMap  map[addr.Addr]*Conn
	queueChan chan addr.Addr
	queueMu   *sync.Mutex
	closed    bool
	closedMu  *sync.RWMutex
	banList   map[addr.IP]bool
	banMu     *sync.RWMutex
}

func New(laddr addr.Addr) (*Node, error) {
	c, err := net.ListenUDP(laddr.Network(), laddr.ToUDPAddr())
	if err != nil {
		return nil, err
	}
	node := &Node{
		conn:      c,
		conns:     map[addr.Addr]*Conn{},
		connsMu:   &sync.RWMutex{},
		queueMap:  map[addr.Addr]*Conn{},
		queueChan: make(chan addr.Addr, maxQueueSize),
		queueMu:   &sync.Mutex{},
		closed:    false,
		closedMu:  &sync.RWMutex{},
		banList:   map[addr.IP]bool{},
		banMu:     &sync.RWMutex{},
	}
	go node.work()
	return node, nil
}

func (node *Node) work() {
	retry := 0
	for {
		if node.isClosed() {
			return
		}
		buffer := make([]byte, maxDatagramSize)
		n, udpAddr, err := node.conn.ReadFromUDP(buffer)
		if err != nil {
			if retry == maxRetry {
				_ = node.Close()
				return
			}
			time.Sleep(sleepDuration)
			retry++
		}
		retry = 0
		buffer = buffer[:n]
		raddr := addr.UDPToAddr(udpAddr)
		node.connsMu.Lock()
		targetConn := node.conns[raddr]
		if targetConn != nil {
			node.connsMu.Unlock()
			targetConn.receive(buffer)
			continue
		}
		node.banMu.RLock()
		banned := node.banList[raddr.IP()]
		node.banMu.RUnlock()
		if banned {
			node.connsMu.Unlock()
			continue
		}
		node.queueMu.Lock()
		targetConn = node.queueMap[raddr]
		if targetConn != nil {
			targetConn.receive(buffer)
			node.queueMu.Unlock()
			node.connsMu.Unlock()
			continue
		}
		if len(node.queueMap) >= maxQueueSize {
			node.queueMu.Unlock()
			node.connsMu.Unlock()
			continue
		}
		targetConn = newConn(node, raddr)
		node.queueMap[raddr] = targetConn
		node.queueMu.Unlock()
		node.connsMu.Unlock()
		node.queueChan <- raddr
		targetConn.receive(buffer)
	}
}

func (node *Node) isClosed() bool {
	node.closedMu.RLock()
	defer node.closedMu.RUnlock()
	return node.closed
}

func (node *Node) Dial(raddr addr.Addr) (conn.Conn, error) {
	if node.isClosed() {
		return nil, ErrNodeClosed
	}
	node.connsMu.Lock()
	defer node.connsMu.Unlock()
	if node.conns[raddr] != nil {
		return nil, ErrDuplicateTuple
	}
	c := newConn(node, raddr)
	node.conns[raddr] = c
	return c, nil
}

func (node *Node) Accept() (conn.Conn, error) {
	if node.isClosed() {
		return nil, ErrNodeClosed
	}
	for {
		select {
		case next := <-node.queueChan:
			node.connsMu.Lock()
			defer node.connsMu.Unlock()
			node.queueMu.Lock()
			defer node.queueMu.Unlock()
			c := node.queueMap[next]
			delete(node.queueMap, next)
			node.conns[next] = c
			return c, nil
		case <-time.Tick(timeoutDuration):
			if node.isClosed() {
				return nil, ErrNodeClosed
			}
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
	defer node.closedMu.Unlock()
	node.closed = true
	return node.conn.Close()
}

func (node *Node) Addr() addr.Addr {
	return addr.UDPToAddr(node.conn.LocalAddr().(*net.UDPAddr))
}

func (node *Node) send(b []byte, raddr addr.Addr) error {
	retry := 0
a:
	_, err := node.conn.WriteToUDP(b, raddr.ToUDPAddr())
	if err != nil {
		nerr, ok := err.(net.Error)
		if ok {
			if nerr.Temporary() {
				if retry == maxRetry {
					return err
				}
				time.Sleep(sleepDuration)
				retry++
				goto a
			}
		}
		return err
	}
	return nil
}
