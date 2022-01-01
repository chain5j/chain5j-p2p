// Package p2ptls
//
// @author: xwc1125
package p2ptls

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"

	ci "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/sec"
)

// ProtocolID 协ID
const ProtocolID = "/tls/1.0.0"

var (
	_ sec.SecureTransport = new(Transport)
)

// Transport 安全通信会话
type Transport struct {
	identity *Identity

	localPeer peer.ID
	privKey   ci.PrivKey
}

func New(cert *tls.Certificate, caRoots [][]byte) (*Transport, error) {
	privateKey := cert.PrivateKey
	key, _, err := ci.KeyPairFromStdKey(privateKey)
	id, err := peer.IDFromPrivateKey(key)
	if err != nil {
		return nil, err
	}
	t := &Transport{
		localPeer: id,
		privKey:   key,
	}

	identity, err := NewIdentity(cert, caRoots)
	if err != nil {
		return nil, err
	}
	t.identity = identity
	return t, nil
}

// SecureInbound runs the TLS handshake as a server.
func (t *Transport) SecureInbound(ctx context.Context, insecure net.Conn) (sec.SecureConn, error) {
	config, keyCh := t.identity.ConfigForAny()
	cs, err := t.handshake(ctx, tls.Server(insecure, config), keyCh)
	if err != nil {
		insecure.Close()
	}
	return cs, err
}

func (t *Transport) handshake(ctx context.Context, tlsConn *tls.Conn, keyCh <-chan ci.PubKey) (sec.SecureConn, error) {
	// There's no way to pass a context to tls.Conn.Handshake().
	// See https://github.com/golang/go/issues/18482.
	// Close the connection instead.
	select {
	case <-ctx.Done():
		tlsConn.Close()
	default:
	}

	done := make(chan struct{})
	var wg sync.WaitGroup

	// Ensure that we do not return before
	// either being done or having a context
	// cancellation.
	defer wg.Wait()
	defer close(done)

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-done:
		case <-ctx.Done():
			tlsConn.Close()
		}
	}()

	if err := tlsConn.Handshake(); err != nil {
		// if the context was canceled, return the context error
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}

	// Should be ready by this point, don't block.
	var remotePubKey ci.PubKey
	select {
	case remotePubKey = <-keyCh:
	default:
	}
	if remotePubKey == nil {
		return nil, errors.New("go-libp2p-tls BUG: expected remote pub key to be set")
	}

	conn, err := t.setupConn(tlsConn, remotePubKey)
	if err != nil {
		// if the context was canceled, return the context error
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}
	return conn, nil
}

// SecureOutbound runs the TLS handshake as a client.
func (t *Transport) SecureOutbound(ctx context.Context, insecure net.Conn, p peer.ID) (sec.SecureConn, error) {
	config, keyCh := t.identity.ConfigForPeer(p)
	tlsConn := tls.Client(insecure, config)
	cs, err := t.handshake(ctx, tlsConn, keyCh)
	if err != nil {
		insecure.Close()
	}
	return cs, err
}

// setupConn ...
func (t *Transport) setupConn(tlsConn *tls.Conn, remotePubKey ci.PubKey) (sec.SecureConn, error) {
	remotePeerID, err := peer.IDFromPublicKey(remotePubKey)
	if err != nil {
		return nil, err
	}

	return &conn{
		Conn:         tlsConn,
		localPeer:    t.localPeer,
		privKey:      t.privKey,
		remotePeer:   remotePeerID,
		remotePubKey: remotePubKey,
	}, nil
}
