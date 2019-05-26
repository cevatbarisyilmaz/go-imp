package udp

import (
	"encoding/binary"
	"math"
	"net"
	"time"
)

const sleepDuration = time.Millisecond / 2
const maxRetry = 7

const (
	smallMessagePack byte = iota
	mediumMessagePack
	largeMessagePack
	extraLargeMessagePack
	acknowledgePack
	requestPack
)

const maxSmallMessageSize = uint64(math.MaxUint8) + 1
const maxMediumMessageSize = uint64(math.MaxUint16) + 1
const maxLargeMessageSize = uint64(math.MaxInt32) + 1

const smallMessageBufferSize = 1
const mediumMessageBufferSize = 2
const largeMessageBufferSize = 4
const extraLargeMessageBufferSize = 8

const maxMessagePerDatagram = 1024

type imp interface {
	work() error
	receive([]byte)
}

type writerImp struct {
	conn *UDPConn
	c    chan struct {
		payload []byte
		time    time.Time
	}
	msg []byte
	id  byte
}

func (i *writerImp) work() error {
	var mainErr error
	i.conn.registerImp(i)
	messageCount := uint64(len(i.msg) / maxMessagePerDatagram)
	var messageCountBufferSize int
	var pack byte
	if messageCount <= maxSmallMessageSize {
		messageCountBufferSize = smallMessageBufferSize
		pack = smallMessagePack
	} else if messageCount <= maxMediumMessageSize {
		messageCountBufferSize = mediumMessageBufferSize
		pack = mediumMessagePack
	} else if messageCount <= maxLargeMessageSize {
		messageCountBufferSize = largeMessageBufferSize
		pack = largeMessagePack
	} else {
		messageCountBufferSize = extraLargeMessageBufferSize
		pack = extraLargeMessagePack
	}
	var messages [][]byte
	for index := uint64(0); index < messageCount; index++ {
		buffer := make([]byte, 8)
		binary.BigEndian.PutUint64(buffer, index)
		message := []byte{pack, i.id}
		message = append(message, buffer[:messageCountBufferSize]...)
		if index == 0 {
			binary.BigEndian.PutUint64(buffer, messageCount-1)
			message = append(message, buffer[:messageCountBufferSize]...)
		}
		message = append(message, i.msg[index*(uint64(len(i.msg))/messageCount):(index+1)*(uint64(len(i.msg))/messageCount)]...)
		messages = append(messages, message)
	}
	initialTime := time.Now()
	for c := 0; c < maxRetry; c++ {
		for _, message := range messages {
			err := i.sendPack(message)
			if err != nil {
				mainErr = err
			}
		}
		select {
		case msg := <-i.c:
			switch msg.payload[0] {
			case acknowledgePack:
				if msg.payload[1] == i.id {
					i.conn.updateRtt(msg.time.Sub(initialTime))
					return nil
				}
			case requestPack:
				if msg.payload[1] == i.id {
					buffer := msg.payload[2:]
					until := len(buffer) / messageCountBufferSize
					for index := 0; index < until; index++ {
						temp := buffer[index*messageCountBufferSize : (index+1)*messageCountBufferSize]
						temp2 := make([]byte, 8-len(temp))
						temp = append(temp2, temp...)
						msgReq := binary.BigEndian.Uint64(temp)
						err := i.sendPack(messages[msgReq])
						if err != nil {
							mainErr = err
						}
					}
					i.conn.updateRtt(msg.time.Sub(initialTime))
				}
			}
		case <-time.Tick(i.conn.getRtt()):
			break
		}
	}
	return mainErr
}

func (i *writerImp) receive(msg []byte) {
	i.c <- struct {
		payload []byte
		time    time.Time
	}{payload: msg, time: time.Now()}
}

func (i *writerImp) sendPack(pack []byte) error {
	var mainErr error
	for d := 0; d <= maxRetry; d++ {
		n, err := i.conn.conn.WriteToUDP(pack, i.conn.raddr)
		if n != len(pack) {
			if err != nil {
				mainErr = err
				if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
					time.Sleep(sleepDuration * time.Duration(d))
				}
			}
		} else {
			return nil
		}
	}
	return mainErr
}

type readerImp struct {
	id    byte
	conn  *UDPConn
	c     chan []byte
	packs map[uint64][]byte
	size  uint64
	msg   chan []byte
}

func (i *readerImp) work() error {
	for uint64(len(i.packs)) < i.size {
		select {
		case pkg := <-i.c:
			if pkg[0] <= extraLargeMessagePack && pkg[1] == i.id {

			}
		}
	}
	return nil
}

func (i *readerImp) receive(b []byte) {
	i.c <- b
}
