// Package p2ptls
//
// @author: xwc1125
package p2ptls

import (
	"crypto/x509"
	"errors"
	"fmt"
)

// CaTrustRoot CA可信根
type CaTrustRoot struct {
	caTrustRoot *x509.CertPool
}

// NewCaTrustRoot 创建CA可信根
func NewCaTrustRoot(caRoots [][]byte) (*CaTrustRoot, error) {
	// CA证书池
	certPool, err := getCerPool(caRoots)
	if err != nil {
		return nil, err
	}
	return &CaTrustRoot{
		caTrustRoot: certPool,
	}, nil
}

func getCerPool(caRoots [][]byte) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	for _, root := range caRoots {
		ok := certPool.AppendCertsFromPEM([]byte(root))
		if !ok {
			return nil, errors.New("failed to parse root certificate")
		}
	}
	return certPool, nil
}

// RootsPool 获取证书池
func (r *CaTrustRoot) RootsPool() *x509.CertPool {
	return r.caTrustRoot
}

// AddCert 添加CA证书
func (r *CaTrustRoot) AddCert(root *x509.Certificate) {
	r.caTrustRoot.AddCert(root)
}

// AppendCertsFromPEM 添加pem证书
func (r *CaTrustRoot) AppendCertsFromPEM(rootPem []byte) bool {
	return r.caTrustRoot.AppendCertsFromPEM(rootPem)
}

// RefreshCARootFromPem 更新CA可信根
func (r *CaTrustRoot) RefreshCARootFromPem(rootsPem [][]byte) bool {
	cerPool, err := getCerPool(rootsPem)
	if err != nil {
		return false
	}
	*r.caTrustRoot = *cerPool
	return true
}

// VerifyCert 验证证书
func (r *CaTrustRoot) VerifyCert(cert *x509.Certificate) error {
	if cert == nil {
		return fmt.Errorf("cert is nil")
	}
	if _, err := cert.Verify(x509.VerifyOptions{Roots: r.caTrustRoot}); err != nil {
		return fmt.Errorf("certificate verify failed: %s", err)
	}
	return nil
}
