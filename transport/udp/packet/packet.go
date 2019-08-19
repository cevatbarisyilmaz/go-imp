package packet

import (
	"encoding/binary"
	"errors"
	"math"
)

const (
	DataPacket byte = iota
	AckPacket
	ReqPacket
)

const maxDataPerDatagram = 1024

var ErrUnknownPacketType = errors.New("unknown packet type")
var ErrSmallBuffer = errors.New("buffer is empty or too small")
var ErrMsgTooLarge = errors.New("message is too large for an IMP message")
var ErrMsgIsEmpty = errors.New("message is nil or empty")
var ErrBrokenMsg = errors.New("message is broken")

type Packet struct {
	PacketType   byte
	MsgID        byte
	PacketID     uint16
	LastPacketID uint16
	Data         []byte
	ReqPackets   []uint16
}

func (p *Packet) Serialize() ([]byte, error) {
	buffer := []byte{p.PacketType, p.MsgID}
	switch p.PacketType {
	case DataPacket:
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, p.PacketID)
		buffer = append(buffer, b...)
		if p.PacketID == 0 {
			binary.BigEndian.PutUint16(b, p.LastPacketID)
			buffer = append(buffer, b...)
		}
		buffer = append(buffer, p.Data...)
	case AckPacket:
	case ReqPacket:
		for _, r := range p.ReqPackets {
			b := make([]byte, 2)
			binary.BigEndian.PutUint16(b, r)
			buffer = append(buffer, b...)
		}
	default:
		return nil, ErrUnknownPacketType
	}
	return buffer, nil
}

func (p *Packet) MustSerialize() []byte {
	b, _ := p.Serialize()
	return b
}

func Deserialize(b []byte) (*Packet, error) {
	if b == nil || len(b) < 2 {
		return nil, ErrSmallBuffer
	}
	p := &Packet{PacketType: b[0], MsgID: b[1]}
	switch p.PacketType {
	case DataPacket:
		if len(b) < 5 {
			return nil, ErrSmallBuffer
		}
		p.PacketID = binary.BigEndian.Uint16(b[2:4])
		if p.PacketID == 0 {
			if len(b) < 7 {
				return nil, ErrSmallBuffer
			}
			p.LastPacketID = binary.BigEndian.Uint16(b[4:6])
			p.Data = b[6:]
		} else {
			p.Data = b[4:]
		}
	case AckPacket:
	case ReqPacket:
		if len(b) < 4 {
			return nil, ErrSmallBuffer
		}
		if len(b)%2 == 1 {
			return nil, ErrBrokenMsg
		}
		for i := 2; i < len(b); i += 2 {
			p.ReqPackets = append(p.ReqPackets, binary.BigEndian.Uint16(b[i:i+2]))
		}
	default:
		return nil, ErrUnknownPacketType
	}
	return p, nil
}

func InitPackets(msgID byte, data []byte) ([]*Packet, error) {
	if data == nil || len(data) == 0 {
		return nil, ErrMsgIsEmpty
	}
	if len(data) > maxDataPerDatagram*(math.MaxInt8+1) {
		return nil, ErrMsgTooLarge
	}
	lastPacketID := uint16(math.Ceil(float64(len(data))/maxDataPerDatagram) - 1)
	packetCount := int(lastPacketID) + 1
	var packets []*Packet
	for index := uint16(0); index <= lastPacketID; index++ {
		packets = append(packets, &Packet{
			PacketType: DataPacket,
			MsgID:      msgID,
			PacketID:   index,
			Data:       data[int(index)*(len(data)/packetCount) : (int(index)+1)*(len(data))/packetCount],
		})
	}
	packets[0].LastPacketID = lastPacketID
	return packets, nil
}

func MustInitPackets(msgID byte, data []byte) []*Packet {
	p, _ := InitPackets(msgID, data)
	return p
}
