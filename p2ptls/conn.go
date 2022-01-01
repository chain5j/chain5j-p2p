// Package p2ptls
//
// @author: xwc1125
package p2ptls

import (
	"crypto/tls"

	ci "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/sec"
)

var (
	_ sec.SecureConn = new(conn)
)

type conn struct {
	*tls.Conn

	localPeer peer.ID
	privKey   ci.PrivKey

	remotePeer   peer.ID
	remotePubKey ci.PubKey
}

// LocalPeer 本地的p2p的ID
func (c *conn) LocalPeer() peer.ID {
	return c.localPeer
}

// LocalPrivateKey 本地的p2p的私钥
func (c *conn) LocalPrivateKey() ci.PrivKey {
	return c.privKey
}

// RemotePeer 远程的p2p的ID
func (c *conn) RemotePeer() peer.ID {
	return c.remotePeer
}

// RemotePublicKey 远程的p2p的公钥
func (c *conn) RemotePublicKey() ci.PubKey {
	return c.remotePubKey
}
