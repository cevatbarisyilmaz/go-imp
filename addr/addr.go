package addr

import (
	"encoding/binary"
	"net"
)

type Addr [19]byte
type IP [16]byte
type Port [2]byte
type Proto byte
type IPv bool

const (
	ProtoIP Proto = iota
	ProtoUDP
	ProtoTCP
)

const (
	Ipv4 IPv = false
	Ipv6 IPv = true
)

func (addr Addr) IP() IP {
	var i IP
	copy(i[:], addr[:16])
	return i
}

func (addr Addr) Port() Port {
	var p Port
	copy(p[:], addr[16:18])
	return p
}

func (addr Addr) Proto() Proto {
	return Proto(addr[18])
}

func (addr Addr) IPv() IPv {
	return addr.IP().IPv()
}

func (addr Addr) ToTCPAddr() *net.TCPAddr {
	return &net.TCPAddr{
		IP:   addr.IP().ToNetIP(),
		Port: addr.Port().ToInt(),
	}
}

func (addr Addr) ToUDPAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   addr.IP().ToNetIP(),
		Port: addr.Port().ToInt(),
	}
}

func (addr Addr) ToIPAddr() *net.IPAddr {
	return &net.IPAddr{
		IP: addr.IP().ToNetIP(),
	}
}

func (addr Addr) Compatible(o Addr) bool {
	return addr.Proto() == o.Proto() && addr.IPv() == o.IPv()
}

func (addr Addr) Network() string {
	switch addr.Proto() {
	case ProtoTCP:
		return "tcp"
	case ProtoUDP:
		return "udp"
	case ProtoIP:
		return "ip"
	default:
		return "<invalid>"
	}
}
func (addr Addr) String() string {
	switch addr.Proto() {
	case ProtoTCP:
		return addr.ToTCPAddr().String()
	case ProtoUDP:
		return addr.ToUDPAddr().String()
	case ProtoIP:
		return addr.ToIPAddr().String()
	default:
		return "<invalid>"
	}
}

func TCPToAddr(tcpAddr *net.TCPAddr) Addr {
	var addr Addr
	copy(addr[0:16], tcpAddr.IP.To16())
	binary.BigEndian.PutUint16(addr[16:18], uint16(tcpAddr.Port))
	addr[18] = byte(ProtoTCP)
	return addr
}

func UDPToAddr(udpAddr *net.UDPAddr) Addr {
	var addr Addr
	copy(addr[0:16], udpAddr.IP.To16())
	binary.BigEndian.PutUint16(addr[16:18], uint16(udpAddr.Port))
	addr[18] = byte(ProtoTCP)
	return addr
}

func NewAddr(ip IP, port Port, proto Proto) Addr {
	var addr Addr
	copy(addr[0:16], ip[:])
	copy(addr[16:18], port[:])
	addr[18] = byte(proto)
	return addr
}

func (ip IP) ToNetIP() net.IP {
	return ip[:]
}

func (ip IP) IPv() IPv {
	if ip.ToNetIP().To4() == nil {
		return Ipv6
	}
	return Ipv4
}

func NetIPToIP(i net.IP) IP {
	var ip [16]byte
	copy(ip[:], i.To16())
	return ip
}

func (port Port) ToUint16() uint16 {
	return binary.BigEndian.Uint16(port[:])
}

func (port Port) ToInt() int {
	return int(port.ToUint16())
}

func Uint16ToPort(p uint16) Port {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], p)
	return Port(b)
}

func IntToPort(p int) Port {
	return Uint16ToPort(uint16(p))
}
