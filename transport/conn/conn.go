package transport

import (
	"github.com/cevatbarisyilmaz/go-imp/addr"
)

type Conn interface {
	Read() ([]byte, error)
	Write([]byte) error
	Close() error
	LocalAddr() addr.Addr
	RemoteAddr() addr.Addr
}
