// Package p2p
//
// @author: xwc1125
package p2p

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/chain5j/chain5j-protocol/models"
	"github.com/chain5j/chain5j-protocol/protocol"
	"github.com/chain5j/logger"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

var (
	errStreamNotFound = errors.New("stream for peer not found")
)

// streamPool net.Stream 管理
type p2pStreamManager struct {
	log     logger.Logger
	ctx     context.Context
	config  protocol.Config
	streams *sync.Map
	p2pNode *p2pNode
}

func newStreamManager(ctx context.Context, config protocol.Config, p2pNode *p2pNode) (*p2pStreamManager, error) {
	return &p2pStreamManager{
		log:     logger.New("p2p"),
		ctx:     ctx,
		config:  config,
		streams: new(sync.Map),
		p2pNode: p2pNode,
	}, nil
}

// addStream 添加流
func (sp *p2pStreamManager) addStream(s network.Stream) error {
	stream := newStream(sp.ctx, s, sp.p2pNode)
	if sp.p2pNode.isMetrics(3) {
		sp.log.Debug("add stream", "remotePeerId", stream.peerId)
	}
	if _, ok := sp.streams.Load(stream.peerId); ok {
		// cached, _ := val.(*p2pStream)
		// if sp.p2pNode.isMetrics(3) {
		// 	sp.log.Debug("stream manager add stream", "peerId", cached.peerId.Pretty())
		// }
		// 删除旧数据
		sp.delStream(stream.peerId)
	}

	sp.streams.Store(stream.peerId, stream)

	return nil
}

func (sp *p2pStreamManager) stop() {
	sp.streams.Range(func(k, v interface{}) bool {
		if s, ok := v.(*p2pStream); ok {
			s.close()
		}
		return true
	})
}

// delStream 删除流
func (sp *p2pStreamManager) delStream(id peer.ID) error {
	if sp.p2pNode.isMetrics(3) {
		sp.log.Debug("del stream", "peerId", id.Pretty())
	}
	if val, ok := sp.streams.Load(id); ok {
		if cached, ok := val.(*p2pStream); ok {
			sp.streams.Delete(cached.peerId)
			cached.close()
		}
	}

	return nil
}

// streamForPeer 根据peerId获取流对象
func (sp *p2pStreamManager) streamForPeer(id peer.ID) (*p2pStream, error) {
	if v, ok := sp.streams.Load(id); ok {
		cached, _ := v.(*p2pStream)
		return cached, nil
	}

	s, err := sp.p2pNode.host.NewStream(sp.p2pNode.ctx, id, ProtocolID)
	if err != nil {
		return nil, err
	}

	if err = sp.addStream(s); err != nil {
		return nil, err
	}

	if v, ok := sp.streams.Load(id); ok {
		cached, _ := v.(*p2pStream)
		return cached, nil
	}

	return nil, errStreamNotFound
}

// send 发送消息
func (sp *p2pStreamManager) send(msg *models.P2PMessage, id peer.ID) error {
	fmt.Println("p2pStreamManager send", id.Pretty())
	s, err := sp.streamForPeer(id)
	if err != nil {
		sp.log.Error("stream for peer err", "err", err)
		return err
	}

	if err := s.send(msg); err != nil {
		sp.delStream(id)
		sp.log.Error("send message err", "err", err)
		return err
	}

	return nil
}
