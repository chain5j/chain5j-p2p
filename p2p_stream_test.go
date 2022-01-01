// Package p2p
//
// @author: xwc1125
package p2p

import (
	"bufio"
	"github.com/libp2p/go-libp2p-core/network"
	"testing"
)

func TestStream(t *testing.T) {
	var s network.Stream
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	_ = rw
}
