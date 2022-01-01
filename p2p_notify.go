// Package p2p
//
// @author: xwc1125
package p2p

import (
	"sync/atomic"
	"time"

	"github.com/chain5j/chain5j-protocol/models"
	"github.com/chain5j/logger"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/multiformats/go-multiaddr"
)

var (
	_ network.Notifiee = new(p2pNotify)
)

// p2pNotify 消息
type p2pNotify struct {
	log     logger.Logger
	p2pNode *p2pNode
}

// Listen 监听
func (n *p2pNotify) Listen(net network.Network, ma multiaddr.Multiaddr) {}

// ListenClose 监听关闭
func (n *p2pNotify) ListenClose(net network.Network, ma multiaddr.Multiaddr) {}

// Connected 连接上
func (n *p2pNotify) Connected(net network.Network, conn network.Conn) {
	id := conn.RemotePeer()
	if n.p2pNode.existPeer(id) {
		return
	}
	// 大于最大链接数量
	count := atomic.LoadInt32(&n.p2pNode.peerCount)
	if count >= n.p2pNode.config.P2PConfig().MaxPeers {
		n.log.Debug("too many peers connected", "count", count)
		go n.p2pNode.dropPeer(id)
		return
	}

	atomic.AddInt32(&n.p2pNode.peerCount, 1)

	if n.p2pNode.isMetrics(1) {
		n.p2pNode.log.Debug("new peer come in", "peerId", id.Pretty())
	}

	if !n.p2pNode.isAuthorizationNode(id) {
		if n.p2pNode.isMetrics(2) {
			n.log.Debug("unauthorized peer", "peerId", id.Pretty())
		}
		go n.p2pNode.dropPeer(id)
		return
	}

	if n.p2pNode.p2pService != nil {
		// 延迟一秒， 等待stream建立完成发送到协议管理模块
		time.AfterFunc(5*time.Second, func() {
			n.p2pNode.handshakeFeed.Send(P2PToPeerID(id))
		})
	}
}

// Disconnected 断开连接
func (n *p2pNotify) Disconnected(net network.Network, conn network.Conn) {
	remotePeerId := conn.RemotePeer().Pretty()

	if n.p2pNode.isMetrics(2) {
		n.p2pNode.log.Debug("disconnect peer", "peerId", remotePeerId)
	}

	if n.p2pNode.p2pService != nil {
		n.p2pNode.peerDropFeed.Send(models.P2PID(remotePeerId))
	}

	atomic.AddInt32(&n.p2pNode.peerCount, -1)
}

// OpenedStream 打开流
func (n *p2pNotify) OpenedStream(net network.Network, stream network.Stream) {
	if n.p2pNode.isMetrics(3) {
		protocolID1 := stream.Protocol()
		n.log.Debug("open stream1", "peerId", stream.Conn().RemotePeer().Pretty(), "streamId", stream.ID(), "protocolID1", string(protocolID1))
	}
	stream.SetProtocol(ProtocolID)
	if n.p2pNode.isMetrics(3) {
		protocolID := stream.Protocol()
		n.log.Debug("open stream", "peerId", stream.Conn().RemotePeer().Pretty(), "streamId", stream.ID(), "protocolID", string(protocolID))
	}
}

// ClosedStream 关闭流
func (n *p2pNotify) ClosedStream(net network.Network, stream network.Stream) {
	protocolID := stream.Protocol()
	if n.p2pNode.isMetrics(3) {
		n.log.Debug("closed stream", "peerId", stream.Conn().RemotePeer().Pretty(), "streamId", stream.ID(), "protocolID", string(protocolID))
	}
	if protocolID == ProtocolID {
		if n.p2pNode.isMetrics(3) {
			n.log.Trace("stream closed, delete from stream manager")
		}
		n.p2pNode.streamManager.delStream(stream.Conn().RemotePeer())
	}
}
