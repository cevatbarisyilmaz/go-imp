package tcp

import (
	"encoding/binary"
	"github.com/cevatbarisyilmaz/go-imp/addr"
	"net"
	"time"
)

const messageSizeLength = 8
const sleepDuration = time.Millisecond * 100
const maxRetry = 7

type Conn struct {
	conn *net.TCPConn
}

func (conn *Conn) Read() ([]byte, error) {
	retry := 0
	sizeBuffer := make([]byte, messageSizeLength)
	sizeLeft := messageSizeLength
	for sizeLeft > 0 {
		buffer := make([]byte, sizeLeft)
		n, err := conn.conn.Read(buffer)
		if n > 0 {
			copy(sizeBuffer[messageSizeLength-sizeLeft:], buffer[:n])
			sizeLeft -= n
		}
		if err != nil {
			if retry == maxRetry {
				return nil, err
			}
			if nerr, ok := err.(net.Error); ok {
				if nerr.Temporary() {
					time.Sleep(sleepDuration * time.Duration(retry+1))
					retry++
				} else {
					return nil, err
				}
			}
		}
	}
	size := binary.BigEndian.Uint64(sizeBuffer)
	retry = 0
	messageBuffer := make([]byte, size)
	messageLeft := size
	for messageLeft > 0 {
		buffer := make([]byte, messageLeft)
		n, err := conn.conn.Read(buffer)
		if n > 0 {
			copy(messageBuffer[size-messageLeft:], buffer[:n])
			messageLeft -= uint64(n)
		}
		if err != nil {
			if retry == maxRetry {
				return nil, err
			}
			if nerr, ok := err.(net.Error); ok {
				if nerr.Temporary() {
					time.Sleep(sleepDuration * time.Duration(retry+1))
					retry++
				}
			} else {
				return nil, err
			}
		}
	}
	return messageBuffer, nil
}

func (conn *Conn) Write(b []byte) error {
	sizeBuffer := make([]byte, messageSizeLength)
	binary.BigEndian.PutUint64(sizeBuffer, uint64(len(b)))
	left := messageSizeLength
	retry := 0
	for left > 0 {
		n, err := conn.conn.Write(sizeBuffer[messageSizeLength-left:])
		left -= n
		if err != nil {
			if retry == maxRetry {
				return err
			}
			if nerr, ok := err.(net.Error); ok {
				if nerr.Temporary() {
					time.Sleep(sleepDuration * time.Duration(retry+1))
					retry++
				}
			} else {
				return err
			}
		}
	}
	left = len(b)
	retry = 0
	for left > 0 {
		n, err := conn.conn.Write(b[len(b)-left:])
		left -= n
		if err != nil {
			if retry == maxRetry {
				return err
			}
			if nerr, ok := err.(net.Error); ok {
				if nerr.Temporary() {
					time.Sleep(sleepDuration * time.Duration(retry+1))
					retry++
				}
			} else {
				return err
			}
		}
	}
	return nil
}

func (conn *Conn) LocalAddr() addr.Addr {
	return addr.TCPToAddr(conn.conn.LocalAddr().(*net.TCPAddr))
}

func (conn *Conn) RemoteAddr() addr.Addr {
	return addr.TCPToAddr(conn.conn.RemoteAddr().(*net.TCPAddr))
}

func (conn *Conn) SetDeadline(t time.Time) error {
	return conn.conn.SetDeadline(t)
}

func (conn *Conn) SetWriteDeadline(t time.Time) error {
	return conn.conn.SetWriteDeadline(t)
}

func (conn *Conn) SetReadDeadline(t time.Time) error {
	return conn.conn.SetReadDeadline(t)
}

func (conn *Conn) Close() error {
	return conn.conn.Close()
}
