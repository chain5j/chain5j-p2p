// Package p2p
//
// @author: xwc1125
package p2p

import (
	"github.com/chain5j/chain5j-protocol/models"
	"github.com/libp2p/go-libp2p-core/peer"
)

// PeerIDtoP2P 将peerId转为peer.ID
func PeerIDtoP2P(peerId models.P2PID) (peer.ID, error) {
	return peer.Decode(string(peerId))
}

// P2PToPeerID 将peer.ID转为models.P2PID
func P2PToPeerID(id peer.ID) models.P2PID {
	return models.P2PID(id.Pretty())
}
