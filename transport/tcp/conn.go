package tcp

import (
	"encoding/binary"
	"errors"
	"github.com/cevatbarisyilmaz/go-imp/addr"
	"net"
	"time"
)

const messageSizeLength = 4
const sleepDuration = time.Millisecond
const maxRetry = 32

type Conn struct {
	conn *net.TCPConn
}

func (conn *Conn) read(size int) ([]byte, error) {
	sizeLeft := size
	errCount := 0
	var message []byte
	for sizeLeft > 0 && errCount < maxRetry {
		buffer := make([]byte, sizeLeft)
		n, err := conn.conn.Read(buffer)
		sizeLeft -= n
		if err != nil {
			errCount++
			if errCount == maxRetry {
				return nil, err
			}
			if nErr, ok := err.(net.Error); ok {
				if nErr.Temporary() {
					time.Sleep(sleepDuration * time.Duration(errCount))
				} else {
					return nil, err
				}
			} else {
				return nil, err
			}
		} else if n == 0 {
			errCount++
			if errCount == maxRetry {
				return nil, errors.New("couldn't read any bytes")
			}
		}
		message = append(message, buffer[:n]...)
	}
	return message, nil
}

func (conn *Conn) Read() ([]byte, error) {
	messageSizeBytes, err := conn.read(messageSizeLength)
	if err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(messageSizeBytes)
	return conn.read(int(size))
}

func (conn *Conn) write(message []byte) error {
	sizeLeft := len(message)
	errCount := 0
	for sizeLeft > 0 && errCount < maxRetry {
		n, err := conn.conn.Write(message)
		sizeLeft -= n
		if err != nil {
			errCount++
			if errCount == maxRetry {
				return err
			}
			if nErr, ok := err.(net.Error); ok {
				if nErr.Temporary() {
					time.Sleep(sleepDuration * time.Duration(errCount))
				} else {
					return err
				}
			} else {
				return err
			}
		} else if n == 0 {
			errCount++
			if errCount == maxRetry {
				return errors.New("couldn't write any bytes")
			}
		}
		message = message[n:]
	}
	return nil
}

func (conn *Conn) Write(message []byte) error {
	buffer := make([]byte, messageSizeLength)
	binary.BigEndian.PutUint32(buffer, uint32(len(message)))
	return conn.write(append(buffer, message...))
}

func (conn *Conn) LocalAddr() addr.Addr {
	return addr.TCPToAddr(conn.conn.LocalAddr().(*net.TCPAddr))
}

func (conn *Conn) RemoteAddr() addr.Addr {
	return addr.TCPToAddr(conn.conn.RemoteAddr().(*net.TCPAddr))
}

func (conn *Conn) Close() error {
	return conn.conn.Close()
}
