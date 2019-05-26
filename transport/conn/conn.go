package transport

import (
	"github.com/cevatbarisyilmaz/go-imp/addr"
	"time"
)

type Conn interface {
	Read() ([]byte, error)
	Write(b []byte) error
	Close() error
	LocalAddr() addr.Addr
	RemoteAddr() addr.Addr
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}
