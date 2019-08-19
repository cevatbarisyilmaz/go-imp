package transport

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"testing"
)

func TestConn(t *testing.T, a, b Conn) {
	const until = 17
	wg := sync.WaitGroup{}
	read := func(conn Conn, messages [][]byte) {
		defer wg.Done()
		for index := 0; index < until; index++ {
			msg, err := conn.Read()
			if err != nil {
				t.Fatal(err)
			}
			messages[index] = msg
		}
		sort.Slice(messages, func(i, j int) bool {
			return len(messages[i]) < len(messages[j])
		})
	}
	wg.Add(2)
	payloadA := make([][]byte, until)
	payloadB := make([][]byte, until)
	messagesA := make([][]byte, until)
	messagesB := make([][]byte, until)
	go read(a, messagesA)
	go read(b, messagesB)
	for index := 0; index < until; index++ {
		for payload, conn := payloadA, a; true; payload, conn = payloadB, b {
			buffer := make([]byte, int(math.Pow(2, float64(index))))
			_, err := rand.Read(buffer)
			if err != nil {
				t.Fatal(err)
			}
			err = conn.Write(buffer)
			if err != nil {
				t.Fatal(err)
			}
			payload[index] = buffer
			if conn == b {
				break
			}
		}
	}
	wg.Wait()
	if !reflect.DeepEqual(payloadA, messagesB) || !reflect.DeepEqual(payloadB, messagesA) {
		fmt.Println(payloadA[0])
		fmt.Println(messagesB[0])
		t.Fatal("failed")
	}
}
