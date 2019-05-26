package transport

import (
	"github.com/cevatbarisyilmaz/go-imp/addr"
	"time"
)

type Conn interface {
	Read() (b []byte, err error)
	Write(b []byte) (n int, err error)
	Close() error
	LocalAddr() addr.Addr
	RemoteAddr() addr.Addr
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}
