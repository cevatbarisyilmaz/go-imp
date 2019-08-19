package udp

import (
	"errors"
	"github.com/cevatbarisyilmaz/go-imp/addr"
	"github.com/cevatbarisyilmaz/go-imp/transport/udp/packet"
	"sync"
	"time"
)

const a = 0.2
const b = 1 - a
const c = 1.1
const startRtt = time.Second * 3
const minRtt = time.Millisecond * 10
const maxRtt = time.Minute
const minMsgSize = 3
const chanSize = 1024

var ErrConnClosed = errors.New("this IMP connection is closed")
var ErrAllIDsAreTaken = errors.New("all available message identifiers are taken")

type Conn struct {
	node         *Node
	raddr        addr.Addr
	queue        chan []byte
	writerIMPs   map[byte]*writerIMP
	writerIPMsMu *sync.RWMutex
	readerIMPs   map[byte]*readerIMP
	readerIMPsMu *sync.RWMutex
	closed       bool
	closedMu     *sync.RWMutex
	rtt          float64
	rttMu        *sync.RWMutex
	msgID        byte
	msgIDMu      *sync.Mutex
}

func newConn(node *Node, raddr addr.Addr) *Conn {
	return &Conn{
		node:         node,
		raddr:        raddr,
		queue:        make(chan []byte, chanSize),
		writerIMPs:   map[byte]*writerIMP{},
		writerIPMsMu: &sync.RWMutex{},
		readerIMPs:   map[byte]*readerIMP{},
		readerIMPsMu: &sync.RWMutex{},
		closed:       false,
		closedMu:     &sync.RWMutex{},
		rtt:          float64(startRtt),
		rttMu:        &sync.RWMutex{},
		msgID:        0,
		msgIDMu:      &sync.Mutex{},
	}
}

func (conn *Conn) getMsgID() byte {
	conn.msgIDMu.Lock()
	defer conn.msgIDMu.Unlock()
	conn.msgID++
	return conn.msgID
}

func (conn *Conn) getRTT() time.Duration {
	conn.rttMu.RLock()
	rtt := conn.rtt * c
	conn.rttMu.RUnlock()
	if rtt < float64(minRtt) {
		rtt = float64(minRtt)
	}
	return time.Duration(rtt)
}

func (conn *Conn) updateRTT(duration time.Duration) {
	conn.rttMu.Lock()
	defer conn.rttMu.Unlock()
	conn.rtt = conn.rtt*b + float64(duration)*a
	if conn.rtt > float64(maxRtt) {
		conn.rtt = float64(maxRtt)
	}
}

func (conn *Conn) receive(b []byte) {
	p, err := packet.Deserialize(b)
	if err != nil {
		return
	}
	switch p.PacketType {
	case packet.DataPacket:
		conn.readerIMPsMu.Lock()
		defer conn.readerIMPsMu.Unlock()
		rimp := conn.readerIMPs[p.MsgID]
		if rimp != nil {
			rimp.receiveChannel <- p
			return
		}
		rimp = newReaderIMP(conn, p.MsgID, make(chan *packet.Packet, chanSize), conn.queue)
		conn.readerIMPs[p.MsgID] = rimp
		rimp.receiveChannel <- p
		go func() {
			_ = rimp.work()
			conn.readerIMPsMu.Lock()
			defer conn.readerIMPsMu.Unlock()
			delete(conn.readerIMPs, p.MsgID)
		}()
	default:
		conn.writerIPMsMu.RLock()
		defer conn.writerIPMsMu.RUnlock()
		wimp := conn.writerIMPs[p.MsgID]
		if wimp == nil {
			return
		}
		wimp.receiveChannel <- p
	}
}

func (conn *Conn) Read() (b []byte, err error) {
	if conn.isClosed() {
		return nil, ErrConnClosed
	}
	msg := <-conn.queue
	if msg == nil {
		return nil, ErrConnClosed
	}
	return msg, nil
}

func (conn *Conn) Write(b []byte) error {
	if conn.isClosed() {
		return ErrConnClosed
	}
	if b == nil || len(b) == 0 {
		return errors.New("nil or empty data")
	}
	conn.writerIPMsMu.Lock()
	msgID := conn.getMsgID()
	if conn.writerIMPs[msgID] != nil {
		conn.writerIPMsMu.Unlock()
		return ErrAllIDsAreTaken
	}
	wimp := newWriterIMP(conn, msgID, b, make(chan *packet.Packet, chanSize))
	conn.writerIMPs[msgID] = wimp
	conn.writerIPMsMu.Unlock()
	go func() {
		_ = wimp.work()
		conn.writerIPMsMu.Lock()
		defer conn.writerIPMsMu.Unlock()
		delete(conn.writerIMPs, msgID)
	}()
	return nil
}

func (conn *Conn) Close() error {
	conn.closedMu.Lock()
	defer conn.closedMu.Unlock()
	conn.closed = true
	return nil
}

func (conn *Conn) LocalAddr() addr.Addr {
	return conn.node.Addr()
}

func (conn *Conn) RemoteAddr() addr.Addr {
	return conn.raddr
}

func (conn *Conn) send(b []byte) error {
	return conn.node.send(b, conn.raddr)
}

func (conn *Conn) isClosed() bool {
	conn.closedMu.RLock()
	defer conn.closedMu.RUnlock()
	return conn.closed
}
