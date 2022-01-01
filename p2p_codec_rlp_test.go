// Package p2p
//
// @author: xwc1125
package p2p

import (
	"bytes"
	"io"
	"strconv"
	"testing"

	"github.com/chain5j/chain5j-protocol/models"
)

func TestRlpxRW(t *testing.T) {
	var ch = make(chan io.ReadWriter, 10)
	go func() {
		for i := 0; i < 50; i++ {
			data := strconv.Itoa(i)
			msg := models.P2PMessage{
				Type: 1,
				Peer: models.P2PID("QmaTgugc8SXYxwGSGhYv5r5B4k5fGDiTUYrVXztZXiR8kx-" + data),
				Data: []byte(data),
			}
			bufw := new(bytes.Buffer)
			rlpW := newCodecW(bufw)
			if err := rlpW.WriteMsg(&msg); err != nil {
				t.Fatal(err)
			}
			ch <- bufw
		}
	}()

	for {
		select {
		case buf := <-ch:
			rlpR := newCodecR(buf)
			msgR, err := rlpR.ReadMsg()
			if err != nil {
				t.Fatal(err)
			}
			t.Log(msgR.Peer, string(msgR.Data))
		}
	}
}
