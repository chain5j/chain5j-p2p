// Package p2ptls
//
// @author: xwc1125
package p2ptls

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	ci "github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
	"github.com/libp2p/go-libp2p-core/peer"
	"golang.org/x/sys/cpu"
)

const (
	alpn string = "libp2p"
)

type Identity struct {
	config tls.Config
	caRoot *CaTrustRoot
}

func NewIdentity(cert *tls.Certificate, caRoots [][]byte) (*Identity, error) {
	// CA证书池
	caTrustRoot, err := NewCaTrustRoot(caRoots)
	if err != nil {
		return nil, err
	}
	return &Identity{
		config: tls.Config{
			MinVersion:               tls.VersionTLS13,
			PreferServerCipherSuites: preferServerCipherSuites(),
			// 用来控制客户端是否证书和服务器主机名。如果设置为true,则不会校验证书以及证书中的主机名和服务器主机名是否一致。
			InsecureSkipVerify: true,
			ClientAuth:         tls.RequireAnyClientCert,
			Certificates:       []tls.Certificate{*cert}, // 自己的证书链
			// VerifyPeerCertificate:  verifyPeerCertificate,
			// ClientCAs:              certPool,
			NextProtos:             []string{alpn},
			SessionTicketsDisabled: true,
		},
		caRoot: caTrustRoot,
	}, nil
}

func (i *Identity) ConfigForAny() (*tls.Config, <-chan ci.PubKey) {
	return i.ConfigForPeer("")
}

func (i *Identity) ConfigForPeer(remote peer.ID) (*tls.Config, <-chan ci.PubKey) {
	keyCh := make(chan ci.PubKey, 1)
	// We need to check the peer ID in the VerifyPeerCertificate callback.
	// The tls.Config it is also used for listening, and we might also have concurrent dials.
	// Clone it so we can check for the specific peer ID we're dialing here.
	conf := i.config.Clone()
	// We're using InsecureSkipVerify, so the verifiedChains parameter will always be empty.
	// We need to parse the certificates ourselves from the raw certs.
	conf.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		defer close(keyCh)

		chain := make([]*x509.Certificate, len(rawCerts))
		for i := 0; i < len(rawCerts); i++ {
			cert, err := x509.ParseCertificate(rawCerts[i])
			if err != nil {
				return err
			}
			chain[i] = cert
		}

		pubKey, err := i.PubKeyFromCertChain(chain)
		if err != nil {
			return err
		}
		if remote != "" && !remote.MatchesPublicKey(pubKey) {
			peerID, err := peer.IDFromPublicKey(pubKey)
			if err != nil {
				peerID = peer.ID(fmt.Sprintf("(not determined: %s)", err.Error()))
			}
			return fmt.Errorf("peer IDs don't match: expected %s, got %s", remote, peerID)
		}
		keyCh <- pubKey
		return nil
	}
	conf.ClientCAs = i.caRoot.RootsPool()
	return conf, keyCh
}

// We want nodes without AES hardware (e.g. ARM) support to always use ChaCha.
// Only if both nodes have AES hardware support (e.g. x86), AES should be used.
// x86->x86: AES, ARM->x86: ChaCha, x86->ARM: ChaCha and ARM->ARM: Chacha
// This function returns true if we don't have AES hardware support, and false otherwise.
// Thus, ARM servers will always use their own cipher suite preferences (ChaCha first),
// and x86 servers will aways use the client's cipher suite preferences.
func preferServerCipherSuites() bool {
	// Copied from the Go TLS implementation.

	// Check the cpu flags for each platform that has optimized GCM implementations.
	// Worst case, these variables will just all be false.
	var (
		hasGCMAsmAMD64 = cpu.X86.HasAES && cpu.X86.HasPCLMULQDQ
		hasGCMAsmARM64 = cpu.ARM64.HasAES && cpu.ARM64.HasPMULL
		// Keep in sync with crypto/aes/cipher_s390x.go.
		hasGCMAsmS390X = cpu.S390X.HasAES && cpu.S390X.HasAESCBC && cpu.S390X.HasAESCTR && (cpu.S390X.HasGHASH || cpu.S390X.HasAESGCM)

		hasGCMAsm = hasGCMAsmAMD64 || hasGCMAsmARM64 || hasGCMAsmS390X
	)
	return !hasGCMAsm
}

type signedKey struct {
	PubKey    []byte
	Signature []byte
}

func (i *Identity) PubKeyFromCertChain(chain []*x509.Certificate) (ci.PubKey, error) {
	if len(chain) != 1 {
		return nil, errors.New("expected one certificates in the chain")
	}
	cert := chain[0]
	if err := i.caRoot.VerifyCert(cert); err != nil {
		return nil, err
	}

	var (
		pbPubKey *pb.PublicKey
	)

	switch p := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		certKeyPub, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
		if err != nil {
			return nil, err
		}
		pbPubKey = &pb.PublicKey{
			Type: pb.KeyType_ECDSA,
			Data: certKeyPub,
		}
	case *rsa.PublicKey:
		certKeyPub, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
		if err != nil {
			return nil, err
		}
		pbPubKey = &pb.PublicKey{
			Type: pb.KeyType_RSA,
			Data: certKeyPub,
		}
	case *btcec.PublicKey:
		pbPubKey = &pb.PublicKey{
			Type: pb.KeyType_Secp256k1,
			Data: p.SerializeCompressed(),
		}
	default:
		return nil, errors.New("unsupported the p2p pubkey")
	}
	pubKey, err := ci.PublicKeyFromProto(pbPubKey)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling public key failed: %s", err)
	}
	return pubKey, nil
}
