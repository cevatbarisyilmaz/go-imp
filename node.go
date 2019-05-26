package imp

import (
	"crypto/rsa"
	"errors"
	"github.com/cevatbarisyilmaz/imp-go/addr"
	"github.com/cevatbarisyilmaz/imp-go/transport"
	"github.com/cevatbarisyilmaz/imp-go/transport/conn"
	"sync"
	"time"
)

const connChanLen = 7
const deadlineDuration = time.Second * 7

var TransportNodeNotFoundErr = errors.New("no transport node is found for given address")
var NoSuitableTransportNodesErr = errors.New("no local compatible transport node is found for given remote address")
var DeadlinePassedErr = errors.New("deadline passed for accepting new connection")

type Node struct {
	PrivateKey  *rsa.PrivateKey
	addrs       map[addr.Addr]transport.Node
	addrsMu     sync.RWMutex
	connChan    chan conn.Conn
	active      bool
	activeMu    sync.RWMutex
	deadline    time.Time
	dialTimeout time.Duration
	banList     map[addr.IP]bool
	banMu       sync.RWMutex
}

func (node *Node) Start() {
	node.connChan = make(chan conn.Conn, connChanLen)
	node.activeMu.Lock()
	node.active = true
	node.activeMu.Unlock()
	if node.addrs == nil {
		node.addrs = make(map[addr.Addr]transport.Node)
	}
	go func() {
		for {
			node.activeMu.RLock()
			if !node.active {
				node.activeMu.RUnlock()
				return
			}
			node.activeMu.RUnlock()
			node.addrsMu.RLock()
			wg := sync.WaitGroup{}
			for _, transportNode := range node.addrs {
				err := transportNode.SetDeadline(time.Now().Add(deadlineDuration))
				if err != nil {
					continue
				}
				wg.Add(1)
				go func() {
					c, err := transportNode.Accept()
					if err == nil {
						node.connChan <- c
					}
					wg.Done()
				}()
			}
			wg.Wait()
		}
	}()
}

func (node *Node) AddAddr(laddr addr.Addr) error {
	transportNode, err := transport.New(laddr)
	if err != nil {
		return err
	}
	transportNode.SetPrivateKey(node.PrivateKey)
	node.banMu.RLock()
	if node.banList != nil {
		for ip := range node.banList {
			transportNode.Ban(ip)
		}
	}
	node.banMu.RUnlock()
	node.addrsMu.Lock()
	defer node.addrsMu.Unlock()
	if node.addrs == nil {
		node.addrs = make(map[addr.Addr]transport.Node)
	}
	node.addrs[laddr] = transportNode
	return nil
}

func (node *Node) RemoveAddr(laddr addr.Addr) error {
	node.addrsMu.Lock()
	defer node.addrsMu.Unlock()
	transportNode := node.addrs[laddr]
	if transportNode == nil {
		return TransportNodeNotFoundErr
	}
	delete(node.addrs, laddr)
	return transportNode.Close()
}

func (node *Node) Dial(raddr addr.Addr) (conn.Conn, error) {
	node.addrsMu.RLock()
	defer node.addrsMu.RUnlock()
	var err error
	for laddr, transportNode := range node.addrs {
		if laddr.Compatible(raddr) {
			if node.dialTimeout != time.Duration(0) {
				transportNode.SetDialTimeout(node.dialTimeout)
			}
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
	return nil, NoSuitableTransportNodesErr
}

func (node *Node) Accept() (conn.Conn, error) {
	if node.deadline.Equal(time.Time{}) {
		return <-node.connChan, nil
	}
	if node.deadline.Before(time.Now()) {
		return nil, DeadlinePassedErr
	}
	select {
	case <-time.Tick(node.deadline.Sub(time.Now())):
		return nil, DeadlinePassedErr
	case c := <-node.connChan:
		return c, nil
	}
}

func (node *Node) SetAcceptDeadline(deadline time.Time) error {
	node.deadline = deadline
	return nil
}

func (node *Node) SetDialTimeout(timeout time.Duration) {
	node.dialTimeout = timeout
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
	if node.addrs == nil {
		return []addr.Addr{}
	}
	addrs := make([]addr.Addr, 0)
	for a := range node.addrs {
		addrs = append(addrs, a)
	}
	return addrs
}

func (node *Node) Ban(raddr addr.IP) {
	node.banMu.Lock()
	if node.banList == nil {
		node.banList = make(map[addr.IP]bool)
	}
	node.banList[raddr] = true
	for _, transportNode := range node.addrs {
		transportNode.Ban(raddr)
	}
	node.banMu.Unlock()
}
