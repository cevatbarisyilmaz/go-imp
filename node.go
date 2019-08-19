package imp

import (
	"errors"
	"github.com/cevatbarisyilmaz/go-imp/addr"
	"github.com/cevatbarisyilmaz/go-imp/transport"
	conn "github.com/cevatbarisyilmaz/go-imp/transport/conn"
	"sync"
	"time"
)

const connChanSize = 7
const timeout = time.Duration(time.Second * 10)

var ErrNoSuitableTransportNodes = errors.New("no local compatible transport node is found for given remote address")
var ErrNodeClosed = errors.New("this IMP node is closed")

type Conn conn.Conn

type Node struct {
	addrs    map[addr.Addr]transport.Node
	addrsMu  *sync.RWMutex
	connChan chan conn.Conn
	active   bool
	activeMu *sync.RWMutex
	banList  map[addr.IP]bool
	banMu    *sync.RWMutex
}

func New() *Node {
	return &Node{
		addrs:    map[addr.Addr]transport.Node{},
		addrsMu:  &sync.RWMutex{},
		connChan: make(chan conn.Conn, connChanSize),
		active:   true,
		activeMu: &sync.RWMutex{},
		banList:  map[addr.IP]bool{},
		banMu:    &sync.RWMutex{},
	}
}

func (node *Node) AddAddr(laddr addr.Addr) error {
	transportNode, err := transport.New(laddr)
	if err != nil {
		return err
	}
	node.addrsMu.Lock()
	defer node.addrsMu.Unlock()
	node.banMu.RLock()
	defer node.banMu.RUnlock()
	for ip := range node.banList {
		transportNode.Ban(ip)
	}
	node.addrs[laddr] = transportNode
	go func() {
		for {
			c, err := transportNode.Accept()
			if err != nil {
				return
			}
			node.connChan <- c
		}
	}()
	return nil
}

func (node *Node) RemoveAddr(laddr addr.Addr) error {
	node.addrsMu.Lock()
	defer node.addrsMu.Unlock()
	transportNode := node.addrs[laddr]
	delete(node.addrs, laddr)
	return transportNode.Close()
}

func (node *Node) Dial(raddr addr.Addr) (Conn, error) {
	node.activeMu.RLock()
	if !node.active {
		node.activeMu.RUnlock()
		return nil, ErrNodeClosed
	}
	node.addrsMu.RLock()
	defer node.addrsMu.RUnlock()
	var err error
	for laddr, transportNode := range node.addrs {
		if laddr.Compatible(raddr) {
			c, e := transportNode.Dial(raddr)
			if e != nil {
				if err == nil {
					err = e
				}
				continue
			}
			return c, nil
		}
	}
	if err != nil {
		return nil, err
	}
	return nil, ErrNoSuitableTransportNodes
}

func (node *Node) Accept() (Conn, error) {
	node.activeMu.RLock()
	if !node.active {
		node.activeMu.RUnlock()
		return nil, ErrNodeClosed
	}
	node.activeMu.RUnlock()
	for {
		select {
		case c := <-node.connChan:
			return c, nil
		case <-time.Tick(timeout):
			node.activeMu.RLock()
			if !node.active {
				node.activeMu.RUnlock()
				return nil, ErrNodeClosed
			}
			node.activeMu.RUnlock()
		}
	}
}

func (node *Node) Close() error {
	node.activeMu.Lock()
	node.active = false
	node.activeMu.Unlock()
	if node.addrs == nil {
		return nil
	}
	var err error
	for _, transportNode := range node.addrs {
		e := transportNode.Close()
		if e != nil && err == nil {
			err = e
		}
	}
	return err
}

func (node *Node) Addrs() []addr.Addr {
	addrs := make([]addr.Addr, 0)
	for a := range node.addrs {
		addrs = append(addrs, a)
	}
	return addrs
}

func (node *Node) Ban(raddr addr.IP) {
	node.addrsMu.RLock()
	defer node.addrsMu.RUnlock()
	node.banMu.Lock()
	defer node.banMu.Unlock()
	node.banList[raddr] = true
	for _, transportNode := range node.addrs {
		transportNode.Ban(raddr)
	}
}
