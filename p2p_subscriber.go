// Package p2p
//
// @author: xwc1125
package p2p

import (
	"github.com/chain5j/chain5j-pkg/event"
	"github.com/chain5j/chain5j-protocol/models"
)

// subscriber 订阅者对象
type subscriber struct {
	feed  event.Feed
	scope event.SubscriptionScope
}

func (n *p2pNode) stopSubscription() {
	n.feeds.Range(func(key, value interface{}) bool {
		if sub, ok := value.(*subscriber); ok {
			sub.scope.Close()
		}
		return true
	})
}

// getSubscriber 根据消息类型获取订阅者
func (n *p2pNode) getSubscriber(msgType uint) *subscriber {
	if value, ok := n.feeds.Load(msgType); ok {
		if sub, ok := value.(*subscriber); ok {
			return sub
		}
	}
	return nil
}

// subscribeMsg 订阅消息
func (s *subscriber) subscribeMsg(ch chan<- *models.P2PMessage) event.Subscription {
	return s.scope.Track(s.feed.Subscribe(ch))
}

// SubscribeMsg 订阅消息
func (n *p2pNode) SubscribeMsg(msgType uint, ch chan<- *models.P2PMessage) event.Subscription {
	var (
		sub *subscriber
	)
	subI, ok := n.feeds.Load(msgType)
	if !ok {
		sub = new(subscriber)
	} else {
		sub = subI.(*subscriber)
	}

	subscription := sub.subscribeMsg(ch)
	n.feeds.Store(msgType, sub)
	return subscription
}

// SubscribeHandshakePeer 订阅握手成功消息
func (n *p2pNode) SubscribeHandshakePeer(ch chan<- models.P2PID) event.Subscription {
	return n.scope.Track(n.handshakeFeed.Subscribe(ch))
}

// SubscribeNewPeer 订阅new peer
func (n *p2pNode) SubscribeNewPeer(ch chan<- models.P2PID) event.Subscription {
	return n.scope.Track(n.newPeerFeed.Subscribe(ch))
}

// SubscribeDropPeer 订阅drop peer
func (n *p2pNode) SubscribeDropPeer(ch chan<- models.P2PID) event.Subscription {
	return n.scope.Track(n.peerDropFeed.Subscribe(ch))
}
