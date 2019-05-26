package udp

import (
	"github.com/cevatbarisyilmaz/go-imp/addr"
	"net"
	"sync"
	"time"
)

type UDPConn struct {
	conn          *net.UDPConn
	raddr         *net.UDPAddr
	readDeadline  time.Time
	writeDeadline time.Time
	imps          map[imp]bool
	impsMu        *sync.RWMutex
	closed        bool
	rtt           float64
	rttMu         *sync.RWMutex
}

func newConn(conn *net.UDPConn, raddr *net.UDPAddr) *UDPConn {
	return &UDPConn{
		conn:         conn,
		raddr:        raddr,
		readDeadline: time.Time{},
		imps:         map[imp]bool{},
		impsMu:       &sync.RWMutex{},
		rtt:          10000,
		rttMu:        &sync.RWMutex{},
	}
}

func (conn *UDPConn) getRtt() time.Duration {
	conn.rttMu.RLock()
	defer conn.rttMu.RUnlock()
	return time.Duration(conn.rtt * 1.1)
}

func (conn *UDPConn) updateRtt(duration time.Duration) {
	conn.rttMu.Lock()
	defer conn.rttMu.Unlock()
	conn.rtt = conn.rtt*0.9 + float64(duration)*0.1
}

func (conn *UDPConn) registerImp(i imp) {
	conn.impsMu.Lock()
	defer conn.impsMu.Unlock()
	conn.imps[i] = true
}

func (conn *UDPConn) unregisterImp(i imp) {
	conn.impsMu.Lock()
	defer conn.impsMu.Unlock()
	delete(conn.imps, i)
}

func (conn *UDPConn) receive(msg []byte) {
	conn.impsMu.RLock()
	defer conn.impsMu.RUnlock()
	for i := range conn.imps {
		i.receive(msg)
	}
}

func (conn *UDPConn) Read() (b []byte, err error) {
	panic("implement me")
}

func (conn *UDPConn) Write(b []byte) (n int, err error) {
	panic("implement me")
}

func (conn *UDPConn) Close() error {
	panic("implement me")
}

func (conn *UDPConn) LocalAddr() addr.Addr {
	panic("implement me")
}

func (conn *UDPConn) RemoteAddr() addr.Addr {
	panic("implement me")
}

func (conn *UDPConn) SetDeadline(t time.Time) error {
	conn.writeDeadline = t
	conn.readDeadline = t
	return nil
}

func (conn *UDPConn) SetReadDeadline(t time.Time) error {
	conn.readDeadline = t
	return nil
}

func (conn *UDPConn) SetWriteDeadline(t time.Time) error {
	conn.writeDeadline = t
	return nil
}
