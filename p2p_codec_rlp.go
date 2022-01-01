// Package p2p
//
// @author: xwc1125
package p2p

import (
	"bytes"
	"errors"
	"io"
	"sync"

	"github.com/chain5j/chain5j-pkg/codec/rlp"
	"github.com/chain5j/chain5j-protocol/models"
)

const (
	maxUint24 = ^uint32(0) >> 8
)

var (
	_ models.MsgReadWriter = new(codec)
)

// codec 在进行p2p发送或接收消息时进行消息编解码
type codec struct {
	w     io.Writer  // 写流
	r     io.Reader  // 读流
	rLock sync.Mutex // 读锁
	wLock sync.Mutex // 写锁
}

func newCodecW(rw io.Writer) models.MsgWriter {
	return &codec{
		w: rw,
	}
}
func newCodecR(rw io.Reader) models.MsgReader {
	return &codec{
		r: rw,
	}
}

// ReadMsg p2p模块消息通信解码
func (r *codec) ReadMsg() (*models.P2PMessage, error) {
	r.rLock.Lock()
	defer r.rLock.Unlock()

	// 读取数据头
	headBuf := make([]byte, 32)
	if _, err := io.ReadFull(r.r, headBuf); err != nil {
		return nil, err
	}

	size := readInt24(headBuf)
	if size > maxUint24 {
		return nil, errors.New("message size overflows uint24")
	}

	frameBuf := make([]byte, size)
	if _, err := io.ReadFull(r.r, frameBuf); err != nil {
		return nil, err
	}

	// 解析数据
	msg := models.P2PMessage{}
	content := bytes.NewReader(frameBuf)
	if err := rlp.Decode(content, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// WriteMsg p2p模块消息编码
func (r *codec) WriteMsg(msg *models.P2PMessage) error {
	r.wLock.Lock()
	r.wLock.Unlock()

	headBuf := make([]byte, 32)
	encBytes, _ := rlp.EncodeToBytes(msg)
	size := uint32(len(encBytes))
	if size > maxUint24 {
		return errors.New("message size overflows uint24")
	}
	putInt24(size, headBuf) // 只占了三个字节，其余留着扩展

	// TODO 两次write 会导致接收解码失败， 未找到具体原因。libp2p的bug？
	wSize, err := r.w.Write(append(headBuf, encBytes...))
	if err != nil {
		return err
	}

	if wSize != 32+len(encBytes) {
		return errors.New("write msg error")
	}

	return nil
}

func putInt24(v uint32, b []byte) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}

func readInt24(b []byte) uint32 {
	return uint32(b[2]) | uint32(b[1])<<8 | uint32(b[0])<<16
}
