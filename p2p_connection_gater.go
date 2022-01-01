// Package p2p
//
// @author: xwc1125
package p2p

import (
	"github.com/chain5j/chain5j-protocol/protocol"
	"github.com/chain5j/logger"
	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/control"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

var (
	_ connmgr.ConnectionGater = (*ConnGater)(nil)
)

// ConnGater p2p网关
// 对出入站的peerId及连接进行权限管控
type ConnGater struct {
	log    logger.Logger
	config protocol.Config
}

// newConnGater 创建网关
func newConnGater(config protocol.Config) *ConnGater {
	return &ConnGater{
		log:    logger.New("p2p"),
		config: config,
	}
}

// InterceptPeerDial 【出站】是否允许向peerId进行拨号
func (cg *ConnGater) InterceptPeerDial(peerId peer.ID) (allow bool) {
	if cg.config.P2PConfig().IsMetrics(3) {
		cg.log.Trace("[out] intercept peer dial", "remotePeerId", peerId.Pretty())
	}
	return true
}

// InterceptAddrDial 是否允许向给定的地址及peerId进行拨号
// 在Network解析了节点地址之后向其拨号之前被调用。
func (cg *ConnGater) InterceptAddrDial(peerId peer.ID, addr multiaddr.Multiaddr) (allow bool) {
	if cg.config.P2PConfig().IsMetrics(3) {
		cg.log.Trace("[out] intercept addr dial", "remotePeerId", peerId.Pretty(), "addr", addr.String())
	}
	return true
}

// InterceptAccept 【入站】是否允许连接入站
// 它会被upgrader或者直接被transport在从socket中接受一个链接之后立即被调用。
// 入站校验第一步
func (cg *ConnGater) InterceptAccept(cm network.ConnMultiaddrs) (allow bool) {
	if cg.config.P2PConfig().IsMetrics(3) {
		cg.log.Trace("[input] intercept addr dial", "localAddr", cm.LocalMultiaddr().String(), "remoteAddr", cm.RemoteMultiaddr().String())
	}
	return true
}

// InterceptSecured 【入站】是否允许给定的连接（已经过身份验证）入站。
// 它会被upgrader执行安全握手之后协商muxer之前调用；
// 或者被transport直接在同一个检查点调用。 包含入站出站
func (cg *ConnGater) InterceptSecured(d network.Direction, peerId peer.ID, cm network.ConnMultiaddrs) (allow bool) {
	if cg.config.P2PConfig().IsMetrics(3) {
		cg.log.Trace("[input] intercept secured", "remotePeerId", peerId.Pretty(), "remoteAddr", cm.RemoteMultiaddr().String(), "net", d.String())
	}
	bl := true
	if d == network.DirInbound {
		// connState := cg.connRecorder.IsConnected(p)
		// if connState {
		//	cg.log.Warn("[Gater] InterceptSecured : %s peer has connected. Ignored.", p.Pretty())
		//	bl = false
		// }
	}
	return bl
}

// InterceptUpgraded 是否允许连接升级
func (cg *ConnGater) InterceptUpgraded(c network.Conn) (allow bool, reason control.DisconnectReason) {
	if cg.config.P2PConfig().IsMetrics(3) {
		cg.log.Trace("[input] intercept upgraded", "remotePeerId", c.RemotePeer().Pretty())
	}
	// p := c.RemotePeer()
	// connState := cg.connRecorder.IsConnected(p)
	// if connState {
	//	logger.Warnf("[Gater] InterceptUpgraded : %s peer has connected. Ignored.", p.Pretty())
	//	return false, 0
	// }
	// cg.connRecorder.AddConn(c.RemotePeer(), c)
	// logger.Debugf("[Gater] InterceptUpgraded : new connection upgraded , remote peer id:%s, remote addr:%s", p.Pretty(), c.RemoteMultiaddr().String())
	return true, 0
}
