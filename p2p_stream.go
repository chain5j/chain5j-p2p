// Package p2p
//
// @author: xwc1125
package p2p

import (
	"context"
	"errors"
	"io"

	"github.com/chain5j/chain5j-protocol/models"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

var (
	errStrNotValid = errors.New("stream not valid")
)

type p2pStream struct {
	ctx    context.Context
	peerId peer.ID

	s    network.Stream
	msgW models.MsgWriter
	msgR models.MsgReader

	p2pNode *p2pNode
}

// newStream p2p数据流处理
func newStream(ctx context.Context, s network.Stream, p2pNode *p2pNode) *p2pStream {
	peerID := s.Conn().RemotePeer()
	ps := &p2pStream{
		ctx:     ctx,
		peerId:  peerID,
		p2pNode: p2pNode,
		s:       s,
		msgW:    newCodecW(s),
		msgR:    newCodecR(s),
	}

	go ps.readDataLoop()
	return ps
}

func (ps *p2pStream) close() {
	ps.s.Reset()
	ps.s.Conn().Close()
}

// send 写数据
func (ps *p2pStream) send(msg *models.P2PMessage) error {
	if ps.p2pNode.isMetrics(2) {
		ps.p2pNode.log.Debug("send msg", "peerId", msg.Peer.String(), "msgType", msg.Type, "msg", string(msg.Data))
	}
	msg.Peer = models.P2PID(ps.p2pNode.id().Pretty())
	if err := ps.msgW.WriteMsg(msg); err != nil {
		return err
	}

	return nil
}

// readData 从p2p stream中读取数据
func (ps *p2pStream) readDataLoop() {
	for {
		msg, err := ps.msgR.ReadMsg()
		switch err {
		case io.EOF:
			ps.p2pNode.log.Error("read stream data", "err", err)
			ps.close()
			return
		case nil:
		default:
			ps.p2pNode.log.Error("read stream data err to reset", "err", err)
			ps.close()
			return
		}
		if ps.p2pNode.isMetrics(2) {
			ps.p2pNode.log.Debug("receive msg", "peerId", msg.Peer.String(), "msgType", msg.Type, "msg", string(msg.Data))
		}
		err = ps.handleMessage(msg)
		if err != nil {
			ps.p2pNode.log.Error("handle p2p msg err", "err", err)
			ps.close()
			return
		}
	}
}

// handleMessage 处理p2p消息
func (ps *p2pStream) handleMessage(msg *models.P2PMessage) error {
	remoteId := ps.s.Conn().RemotePeer().Pretty()
	if remoteId != string(msg.Peer) {
		return errors.New("invalid remote peer Id")
	}
	if ps.p2pNode.p2pService != nil {
		return ps.p2pNode.p2pService.handleMessage(msg)
	}
	return nil
}
