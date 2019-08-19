package transport

import (
	"errors"
	"github.com/cevatbarisyilmaz/go-imp/addr"
	conn "github.com/cevatbarisyilmaz/go-imp/transport/conn"
	"github.com/cevatbarisyilmaz/go-imp/transport/tcp"
	"github.com/cevatbarisyilmaz/go-imp/transport/udp"
)

var UnknownErr = errors.New("unknown protocol")

type Node interface {
	Dial(addr.Addr) (conn.Conn, error)
	Accept() (conn.Conn, error)
	Ban(addr.IP)
	Close() error
	Addr() addr.Addr
}

func New(laddr addr.Addr) (Node, error) {
	switch laddr.Proto() {
	case addr.ProtoTCP:
		return tcp.New(laddr)
	case addr.ProtoUDP:
		return udp.New(laddr)
	default:
		return nil, UnknownErr
	}
}
