// Package p2p
//
// @author: xwc1125
package p2p

import (
	"crypto/tls"
	"encoding/base64"
	"io/ioutil"
	"os"

	"github.com/chain5j/chain5j-protocol/protocol"
	"github.com/chain5j/logger"
	ci "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/tjfoc/gmsm/gmtls"
)

type Certificate struct {
	TlsType     int
	Certificate interface{}
}

// getPrvKey 获取p2p的私钥
// 私钥可能为libp2p的直接私钥，此情况下，必然使用的非tls
// 如果使用的是tls，那么私钥pem和cert的pem
func getPrvKey(log logger.Logger, config protocol.Config) (ci.PrivKey, *Certificate, error) {
	var (
		prvKey ci.PrivKey
		err    error
	)
	if config.P2PConfig().IsTls {
		certBytes, err := ioutil.ReadFile(config.P2PConfig().CertPath)
		if err != nil {
			return nil, nil, err
		}
		keyBytes, err := ioutil.ReadFile(config.P2PConfig().KeyPath)
		if err != nil {
			return nil, nil, err
		}

		gmcert, err := gmtls.X509KeyPair(certBytes, keyBytes)
		if err == nil {
			// gm
			if err != nil {
				return nil, nil, err
			}
			prvKey, _, err = ci.KeyPairFromStdKey(gmcert.PrivateKey)
			if err != nil {
				return nil, nil, err
			}
			return prvKey, &Certificate{
				TlsType:     20,
				Certificate: &gmcert,
			}, nil
		} else {
			cert, err := tls.X509KeyPair(certBytes, keyBytes)
			if err != nil {
				return nil, nil, err
			}
			prvKey, _, err = ci.KeyPairFromStdKey(cert.PrivateKey)
			if err != nil {
				return nil, nil, err
			}
			return prvKey, &Certificate{
				TlsType:     1,
				Certificate: &cert,
			}, nil
		}
	}

	// 使用libp2p的原生私钥
	keyPath := config.P2PConfig().KeyPath
	if len(keyPath) == 0 {
		keyPath = "./p2p/peerkey"
	}
	if prvKeyBytes, _ := os.ReadFile(keyPath); prvKeyBytes != nil {
		// 文件存在
		destKeyBytes, err := base64.StdEncoding.DecodeString(string(prvKeyBytes))
		if err != nil {
			log.Error("base64 decode prvKey err", "err", err)
			return nil, nil, err
		}
		if prvKey, err = ci.UnmarshalPrivateKey(destKeyBytes); err != nil {
			log.Error("unmarshal prvKey err", "err", err)
			return nil, nil, err
		}
	}
	if prvKey == nil {
		// 创建私钥
		prvKey, _, err = ci.GenerateKeyPair(
			ci.ECDSA,
			-1, // RSA时才需要
		)
		if err != nil {
			log.Error("generate new prvKey err", "err", err)
			return nil, nil, err
		}

		if prvKeyBytes, err := ci.MarshalPrivateKey(prvKey); err != nil {
			log.Error("get prvKey bytes err", "err", err)
			return nil, nil, err
		} else {
			os.WriteFile(keyPath, prvKeyBytes, os.ModePerm)
		}
	}
	return prvKey, nil, nil
}
