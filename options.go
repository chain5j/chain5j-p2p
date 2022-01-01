// Package p2p
//
// @author: xwc1125
package p2p

import (
	"fmt"

	"github.com/chain5j/chain5j-protocol/protocol"
)

type option func(f *p2pService) error

func apply(f *p2pService, opts ...option) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(f); err != nil {
			return fmt.Errorf("option apply err:%v", err)
		}
	}
	return nil
}

func WithConfig(config protocol.Config) option {
	return func(f *p2pService) error {
		f.config = config
		return nil
	}
}

func WithAPIs(apis protocol.APIs) option {
	return func(f *p2pService) error {
		f.apis = apis
		return nil
	}
}
