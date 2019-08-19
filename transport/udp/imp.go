package udp

import (
	"errors"
	"github.com/cevatbarisyilmaz/go-imp/transport/udp/packet"
	"time"
)

const ackEnsureMultiplier = 7

type writerIMP struct {
	packets        []*packet.Packet
	receiveChannel chan *packet.Packet
	conn           *Conn
}

func newWriterIMP(conn *Conn, msgID byte, data []byte, receiveChannel chan *packet.Packet) *writerIMP {
	packets := packet.MustInitPackets(msgID, data)
	return &writerIMP{
		packets:        packets,
		receiveChannel: receiveChannel,
		conn:           conn,
	}
}

func (i *writerIMP) work() error {
	packets := i.packets
	updateRTT := true
	for count := 0; count < maxRetry; count++ {
		for _, p := range packets {
			err := i.conn.send(p.MustSerialize())
			if err != nil {
				return err
			}
		}
		referenceTime := time.Now()
		select {
		case reply := <-i.receiveChannel:
			switch reply.PacketType {
			case packet.AckPacket:
				if updateRTT {
					i.conn.updateRTT(time.Now().Sub(referenceTime))
				}
				return nil
			case packet.ReqPacket:
				if updateRTT {
					i.conn.updateRTT(time.Now().Sub(referenceTime))
					updateRTT = false
				}
				packets = make([]*packet.Packet, 0)
				for _, rp := range reply.ReqPackets {
					packets = append(packets, i.packets[int(rp)])
				}
			}
		case <-time.Tick(i.conn.getRTT()):
			updateRTT = false
		}
	}
	return errors.New("write failed")
}

type readerIMP struct {
	receiveChannel chan *packet.Packet
	sendChannel    chan []byte
	conn           *Conn
	msgID          byte
}

func newReaderIMP(conn *Conn, msgID byte, receiveChannel chan *packet.Packet, sendChannel chan []byte) *readerIMP {
	return &readerIMP{
		receiveChannel: receiveChannel,
		sendChannel:    sendChannel,
		conn:           conn,
		msgID:          msgID,
	}
}

func (i *readerIMP) work() error {
	packets := map[uint16][]byte{}
	var receivedHighestPacketID uint16
	var announcedHighestPacketID uint16
	var packetZeroArrived bool
	failure := 0
l:
	for failure < maxRetry {
		select {
		case p := <-i.receiveChannel:
			if p.PacketID == 0 {
				announcedHighestPacketID = p.LastPacketID
				packetZeroArrived = true
			} else if !packetZeroArrived && p.PacketID > receivedHighestPacketID {
				receivedHighestPacketID = p.PacketID
			} else if packetZeroArrived && p.PacketID > announcedHighestPacketID {
				failure++
				continue l
			}
			packets[p.PacketID] = p.Data
			if packetZeroArrived && uint16(len(packets)-1) == announcedHighestPacketID {
				for index := uint16(0); index <= announcedHighestPacketID; index++ {
					if packets[index] == nil {
						failure++
						continue l
					}
				}
				break l
			}
		case <-time.Tick(i.conn.getRTT()):
			failure++
			var until uint16
			var reqPackets []uint16
			if packetZeroArrived {
				until = announcedHighestPacketID
			} else {
				until = receivedHighestPacketID
			}
			for index := uint16(0); index <= until; index++ {
				if packets[index] == nil {
					reqPackets = append(reqPackets, index)
				}
			}
			p := &packet.Packet{
				PacketType: packet.ReqPacket,
				MsgID:      i.msgID,
				ReqPackets: reqPackets,
			}
			if i.conn.send(p.MustSerialize()) != nil {
				failure++
			}
		}
	}
	if failure >= maxRetry {
		return errors.New("read failed")
	}
	var msg []byte
	for index := uint16(0); index <= announcedHighestPacketID; index++ {
		msg = append(msg, packets[index]...)
	}
	i.sendChannel <- msg
	ackPacket := &packet.Packet{
		PacketType: packet.AckPacket,
		MsgID:      i.msgID,
	}
	for failure = 0; failure < maxRetry; failure++ {
		err := i.conn.send(ackPacket.MustSerialize())
		if err != nil {
			return err
		}
		select {
		case <-i.receiveChannel:
			continue
		case <-time.Tick(i.conn.getRTT() * ackEnsureMultiplier):
			return nil
		}
	}
	return errors.New("acknowledgement error")
}
