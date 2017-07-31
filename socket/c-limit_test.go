package socket

import (
	"log"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mh-cbon/dht/kmsg"
)

func TestCLimit(t *testing.T) {

	t.Run("should not exceed query limit", func(t *testing.T) {

		var tcnt int32
		var cnt int32
		concurrency := 5
		concurrencyMax := 12
		concurrencyMin := 4

		alice := New(NewConfig("127.0.0.1:9540"))
		go alice.MustListen(func(msg kmsg.Msg, from *net.UDPAddr) error {
			<-time.After(time.Millisecond * 10)
			return alice.Respond(from, msg.T, kmsg.Return{V: "hi!"})
		})

		bob := NewConcurrent(concurrency, NewConfig("127.0.0.1:9541"))
		go bob.MustListen(nil)

		<-time.After(time.Millisecond)

		done := make(chan bool)
		go func() {
			for {
				go atomic.AddInt32(&tcnt, 1)
				atomic.AddInt32(&cnt, 1)
				bob.Query(makeAddr("127.0.0.1:9540"), "meet", nil, func(kmsg.Msg) {
					atomic.AddInt32(&cnt, -1)
				})
				select {
				case <-time.After(time.Microsecond):
				case <-done:
					return
				}
			}
		}()
		go func() {
			<-time.After(time.Millisecond * 5)
			for {
				if cnt > int32(concurrencyMax) {
					t.Errorf("Exceeded concurrency max=%v wanted=%v got=%v", concurrencyMax, concurrency, cnt)
				} else if cnt < int32(concurrencyMin) {
					t.Errorf("Low concurrency min=%v wanted=%v got=%v", concurrencyMin, concurrency, cnt)
				}
				select {
				case <-time.After(time.Millisecond * 5):
				case <-done:
					return
				}
			}
		}()

		<-time.After(time.Millisecond * 250)
		if tcnt < 20 {
			t.Errorf("Unsufficent query executed wanted>%v got=%v", 50, tcnt)
		}
		if cnt > int32(concurrencyMax) {
			t.Errorf("Exceeded concurrency max=%v wanted=%v got=%v", concurrencyMax, concurrency, cnt)
		}
		done <- true

		alice.Close()
		bob.Close()
		<-time.After(time.Millisecond)
	})
	t.Run("should not exceed query limit2", func(t *testing.T) {
		alice := New(NewConfig("127.0.0.1:9540"))
		i := 10
		go alice.MustListen(func(msg kmsg.Msg, from *net.UDPAddr) error {
			i--
			<-time.After(time.Duration(i) * time.Millisecond)
			return alice.Respond(from, msg.T, kmsg.Return{V: "hi!"})
		})

		bob := NewConcurrent(5, NewConfig("127.0.0.1:9541"))
		go bob.MustListen(nil)

		<-time.After(time.Millisecond)

		go bob.Query(makeAddr("127.0.0.1:9540"), "meet", nil, func(kmsg.Msg) {
			log.Println("timeout 1")
		})
		go bob.Query(makeAddr("127.0.0.1:9540"), "meet", nil, func(kmsg.Msg) {
			log.Println("timeout 2")
		})
		go bob.Query(makeAddr("127.0.0.1:9540"), "meet", nil, func(kmsg.Msg) {
			log.Println("timeout 3")
		})

		<-time.After(time.Millisecond * 250)

		alice.Close()
		bob.Close()
	})

}