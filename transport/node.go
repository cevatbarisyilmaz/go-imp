package transport

import (
	"crypto/rsa"
	"errors"
	"github.com/cevatbarisyilmaz/imp-go/addr"
	"github.com/cevatbarisyilmaz/imp-go/transport/conn"
	"github.com/cevatbarisyilmaz/imp-go/transport/tcp"
	"time"
)

var UnimplementedErr = errors.New("unimplemented protocol")
var UnknownErr = errors.New("unknown protocol")

type Node interface {
	SetPrivateKey(*rsa.PrivateKey)
	Dial(addr.Addr) (conn.Conn, error)
	Accept() (conn.Conn, error)
	Ban(addr.IP)
	SetDeadline(time.Time) error
	SetDialTimeout(time.Duration)
	Close() error
	Addr() addr.Addr
}

func New(laddr addr.Addr) (Node, error) {
	switch laddr.Proto() {
	case addr.ProtoTCP:
		return tcp.New(laddr.ToTCPAddr())
	case addr.ProtoUDP:
		return nil, UnimplementedErr
	case addr.ProtoIP:
		return nil, UnimplementedErr
	default:
		return nil, UnknownErr
	}
}
