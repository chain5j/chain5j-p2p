// Package p2p
//
// @author: xwc1125
package p2p

import (
	"context"
	"fmt"

	"github.com/chain5j/chain5j-pkg/event"
	"github.com/chain5j/chain5j-protocol/models"
	"github.com/chain5j/chain5j-protocol/protocol"
	"github.com/chain5j/logger"
)

var (
	_ protocol.P2PService = new(p2pService)
)

// p2pService p2p 服务
type p2pService struct {
	log    logger.Logger
	ctx    context.Context
	cancel context.CancelFunc

	config protocol.Config
	apis   protocol.APIs

	p2pNode *p2pNode
}

// NewP2P 创建p2p服务
func NewP2P(rootCtx context.Context, opts ...option) (protocol.P2PService, error) {
	ctx, cancel := context.WithCancel(rootCtx)
	service := &p2pService{
		log:    logger.New("p2p"),
		ctx:    ctx,
		cancel: cancel,
	}
	if err := apply(service, opts...); err != nil {
		return nil, err
	}

	p2pNode, err := newP2PNode(ctx, service.config)
	if err != nil {
		service.log.Error("new p2p node err", "err", err)
		return nil, err
	}
	p2pNode.setP2PService(service)

	service.p2pNode = p2pNode

	// 注册API
	registerAPI(service)
	return service, nil
}

// registerAPI 注册API
func registerAPI(service *p2pService) {
	if service.apis != nil {
		service.apis.RegisterAPI([]protocol.API{
			{
				Namespace: "node", // namespace
				Version:   "1.0",
				Service:   newAPI(service),
				Public:    true,
			},
			{
				Namespace: "admin", // namespace
				Version:   "1.0",
				Service:   newP2PAdminApi(service),
				Public:    false,
			},
			{
				Namespace: "net", // namespace
				Version:   "1.0",
				Service:   newNetAPI(service, 2),
				Public:    true,
			},
		})
	}
}

// Start 启动
func (s *p2pService) Start() (err error) {
	go s.p2pNode.start()
	return nil
}

// Stop 停止
func (s *p2pService) Stop() error {
	s.p2pNode.stop()
	s.cancel()
	return nil
}

// NetURL 获取p2p的url
func (s *p2pService) NetURL() string {
	addr := getP2PAddr(s.config)
	return fmt.Sprintf(addr+"/p2p/%s", s.p2pNode.id().Pretty())
}

// RemotePeers 已经连接的节点列表
func (s *p2pService) RemotePeers() []models.P2PID {
	return s.p2pNode.connectedPeers()
}

// P2PInfo peer对象信息
func (s *p2pService) P2PInfo() map[string]*models.P2PInfo {
	return s.p2pNode.peers()
}

// HandshakeSuccess 握手成功
func (s *p2pService) HandshakeSuccess(id models.P2PID) {
	peerId, err := PeerIDtoP2P(id)
	if err != nil {
		s.log.Error("peerId to p2p err", "peerId", id, "err", err)
		return
	}
	s.p2pNode.handshakeSuccess(peerId)
}

// Send 发送消息
func (s *p2pService) Send(peerId models.P2PID, msg *models.P2PMessage) error {
	id, err := PeerIDtoP2P(peerId)
	if err != nil {
		s.log.Error("peerId to p2p err", "peerId", id, "err", err)
		return err
	}
	return s.p2pNode.sendMsg(id, msg)
}

// Id 获取ID
func (s *p2pService) Id() models.P2PID {
	return P2PToPeerID(s.p2pNode.id())
}

// handleMessage 处理消息
func (s *p2pService) handleMessage(msg *models.P2PMessage) error {
	// 获取订阅者
	sub := s.p2pNode.getSubscriber(msg.Type)
	if sub != nil {
		sub.feed.Send(msg)
	}

	return nil
}

// AddPeer 增加peer
func (s *p2pService) AddPeer(peerUrl string) error {
	return s.p2pNode.addPeer(peerUrl)
}

// DropPeer 移除peer
func (s *p2pService) DropPeer(id models.P2PID) error {
	peerId, err := PeerIDtoP2P(id)
	if err != nil {
		s.log.Error("peerId to p2p err", "peerId", id, "err", err)
		return err
	}
	s.p2pNode.dropPeer(peerId)
	return nil
}

// SubscribeMsg 订阅消息
func (s *p2pService) SubscribeMsg(msgType uint, ch chan<- *models.P2PMessage) event.Subscription {
	return s.p2pNode.SubscribeMsg(msgType, ch)
}

// SubscribeHandshakePeer 订阅握手成功消息
func (s *p2pService) SubscribeHandshakePeer(ch chan<- models.P2PID) event.Subscription {
	return s.p2pNode.SubscribeHandshakePeer(ch)
}

// SubscribeNewPeer 订阅new peer
func (s *p2pService) SubscribeNewPeer(ch chan<- models.P2PID) event.Subscription {
	return s.p2pNode.SubscribeNewPeer(ch)
}

// SubscribeDropPeer 订阅drop peer
func (s *p2pService) SubscribeDropPeer(ch chan<- models.P2PID) event.Subscription {
	return s.p2pNode.SubscribeDropPeer(ch)
}
