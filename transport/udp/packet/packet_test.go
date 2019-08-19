package packet_test

import (
	"bytes"
	"github.com/cevatbarisyilmaz/go-imp/transport/udp/packet"
	"math"
	"math/rand"
	"reflect"
	"testing"
)

func TestPacket_Serialize(t *testing.T) {
	packets := []*packet.Packet{
		{PacketType: packet.DataPacket, MsgID: 23, PacketID: 0, LastPacketID: 3, Data: []byte{3, 5, 7}},
		{PacketType: packet.DataPacket, MsgID: 2, PacketID: 3, Data: []byte{2, 2}},
		{PacketType: packet.AckPacket, MsgID: 44},
		{PacketType: packet.ReqPacket, MsgID: 22, ReqPackets: []uint16{4, 5, 12}},
	}
	for _, p := range packets {
		b, err := p.Serialize()
		if err != nil {
			t.Error(err)
			continue
		}
		dp, err := packet.Deserialize(b)
		if err != nil {
			t.Error(err)
			continue
		}
		if !reflect.DeepEqual(p, dp) {
			t.Error(p, "and", dp, "are not deep equal")
			continue
		}
	}
}

func TestInitPackets(t *testing.T) {
	for i := byte(0); i < 8; i++ {
		size := int(math.Pow(2, float64(i)))
		buffer := make([]byte, size)
		_, _ = rand.Read(buffer)
		packets, err := packet.InitPackets(i, buffer)
		if err != nil {
			t.Error(err)
			continue
		}
		var checkBuffer []byte
		for pi, p := range packets {
			if p.PacketID != uint16(pi) {
				t.Error("out of order packet")
			}
			if p.MsgID != i {
				t.Error("wrong message ID")
			}
			if p.PacketType != packet.DataPacket {
				t.Error("wrong packet type")
			}
			if p.PacketID == 0 && p.LastPacketID != uint16(len(packets)-1) {
				t.Error("wrong last packet ID")
			}
			checkBuffer = append(checkBuffer, p.Data...)
		}
		if !bytes.Equal(buffer, checkBuffer) {
			t.Error("wrong data")
		}
	}
}
