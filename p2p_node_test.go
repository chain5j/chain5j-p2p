// Package p2p
//
// @author: xwc1125
package p2p

import (
	"context"
	"github.com/chain5j/chain5j-pkg/util/convutil"
	"github.com/chain5j/chain5j-protocol/mock"
	"github.com/chain5j/chain5j-protocol/models"
	"github.com/chain5j/logger"
	"github.com/chain5j/logger/zap"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"
	"testing"
	"time"
)

func init() {
	zap.InitWithConfig(&logger.LogConfig{
		Console: logger.ConsoleLogConfig{
			Level:    4,
			Modules:  "*",
			ShowPath: false,
			Format:   "",
			UseColor: true,
			Console:  true,
		},
		File: logger.FileLogConfig{},
	})
}

func TestNewP2PNode1(t *testing.T) {
	pNode := getNode(t, models.P2PConfig{
		Host:             "",
		Port:             9545,
		KeyPath:          "./testdata/node1/nodekey",
		CertPath:         "",
		IsTls:            false,
		EnablePermission: false,
		MaxPeers:         MaxPeers,
		StaticNodes:      []string{"/ip4/127.0.0.1/tcp/9546/p2p/QmafFDAXGtW1M2zhsiYPxUtpkYjAgBUpPDPQZRkhgonGrT"},
		CaRoots:          nil,
		Metrics:          true,
		MetricsLevel:     3,
	})
	go pNode.start()
	node2Id, _ := peer.Decode("QmafFDAXGtW1M2zhsiYPxUtpkYjAgBUpPDPQZRkhgonGrT")
	ticker := time.NewTicker(5 * time.Second)
	count := 0
	for {
		select {
		case <-ticker.C:
			count++
			pNode.sendMsg(node2Id, &models.P2PMessage{
				Type: 1,
				Peer: "",
				Data: []byte("I am node1==>" + convutil.ToString(count)),
			})
		}
	}
}
func TestNewP2PNode2(t *testing.T) {
	pNode := getNode(t, models.P2PConfig{
		Host:             "",
		Port:             9546,
		KeyPath:          "./testdata/node2/nodekey",
		CertPath:         "",
		IsTls:            false,
		EnablePermission: false,
		MaxPeers:         MaxPeers,
		StaticNodes:      []string{"/ip4/127.0.0.1/tcp/9545/p2p/QmVy5JASWLUns3Wwe91arP93rPPumkEQ7fQZ4GfpevxKbd"},
		CaRoots:          nil,
		Metrics:          true,
		MetricsLevel:     3,
	})
	go pNode.start()
	node1Id, _ := peer.Decode("QmVy5JASWLUns3Wwe91arP93rPPumkEQ7fQZ4GfpevxKbd")
	ticker := time.NewTicker(5 * time.Second)
	count := 0
	for {
		select {
		case <-ticker.C:
			count++
			pNode.sendMsg(node1Id, &models.P2PMessage{
				Type: 1,
				Peer: "",
				Data: []byte("I am node2==>" + convutil.ToString(count)),
			})
		}
	}
}
func getNode(t *testing.T, p2pConfig models.P2PConfig) *p2pNode {
	mockCtl := gomock.NewController(nil)
	// config
	mockConfig := mock.NewMockConfig(mockCtl)
	// node1: QmQRMq967n6mvXhw4eCY7QrgAYmUPx52tQ1x4mXo8Zwg7p
	// node2: QmaTgugc8SXYxwGSGhYv5r5B4k5fGDiTUYrVXztZXiR8kx
	mockConfig.EXPECT().P2PConfig().Return(p2pConfig).AnyTimes()

	node, err := newP2PNode(context.Background(), mockConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("make host Id: %s\n", node.id())
	return node
}
