// Package p2p
//
// @author: xwc1125
package p2p

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chain5j/chain5j-pkg/util/convutil"
	"github.com/chain5j/chain5j-protocol/mock"
	"github.com/chain5j/chain5j-protocol/models"
	"github.com/chain5j/chain5j-protocol/protocol"
	"github.com/chain5j/logger"
	"github.com/chain5j/logger/zap"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
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

func TestP2PServer1(t *testing.T) {
	var wg sync.WaitGroup

	mockCtl := gomock.NewController(nil)
	mockConfig := mock.NewMockConfig(mockCtl)
	p2pConfig := models.P2PConfig{
		Host:    "",
		Port:    9545,
		KeyPath: "/Users/yijiaren/Workspaces/git/chain5j/chain5j/conf/certs/ssl/server/server-key.pem",
		// KeyPath:          "./testdata/node1/nodekey",
		CertPath:         "/Users/yijiaren/Workspaces/git/chain5j/chain5j/conf/certs/ssl/server/server.pem",
		IsTls:            true,
		EnablePermission: false,
		MaxPeers:         MaxPeers,
		// StaticNodes:      []string{"/ip4/127.0.0.1/tcp/9546/p2p/QmafFDAXGtW1M2zhsiYPxUtpkYjAgBUpPDPQZRkhgonGrT"},
		// StaticNodes: []string{"/ip4/127.0.0.1/tcp/9546/p2p/QmQSmLbF7Q6crFLx3GvzKEmiyeAbnknst1pFUQaqZcc64v"},
		StaticNodes: []string{"/ip4/127.0.0.1/tcp/9546/p2p/QmRqKVoDy23eJtUWJeEHSXeqZFLjgC98AT2yoV4dzqnMZv"},
		CaRoots: []string{
			"/Users/yijiaren/Workspaces/git/chain5j/chain5j/conf/certs/ssl/ca/ca.pem",
			"/Users/yijiaren/Workspaces/git/chain5j/chain5j/conf/certs/ssl2/ca/ca.pem",
			"/Users/yijiaren/Workspaces/git/chain5j/chain5j/conf/certs/ssl2/ca/ca-sub.pem",
			// "/Users/yijiaren/Workspaces/git/chain5j/chain5j/conf/certs/ssl2/ca/ca-all.pem",
			// "/Users/yijiaren/Workspaces/git/chain5j/chain5j/conf/certs/ssl2/server/server-all.pem",
		},
		Metrics:      true,
		MetricsLevel: 2,
	}
	mockConfig.EXPECT().P2PConfig().Return(p2pConfig).AnyTimes()

	service1, err := NewP2P(
		context.Background(),
		WithConfig(mockConfig),
	)
	if err != nil {
		t.Fatal(err)
	}
	go testPeerEvent(service1, &wg, t)

	service1.Start()

	// 接收消息
	go testMsgRecv(t, service1, &wg)

	// 等待连接
	time.Sleep(5 * time.Second)
	if len(p2pConfig.StaticNodes) > 0 {
		for _, node := range p2pConfig.StaticNodes {
			muAddr, err := multiaddr.NewMultiaddr(node)
			if err != nil {
				t.Error(err)
				continue
			}
			info, err := peer.AddrInfoFromP2pAddr(muAddr)
			if err != nil {
				t.Error(err)
				continue
			}
			go sendMsg(service1, models.P2PID(info.ID.Pretty()), &wg)
		}
	}

	wg.Add(1)
	wg.Wait()
}
func TestP2PServer2(t *testing.T) {
	var wg sync.WaitGroup

	mockCtl := gomock.NewController(nil)
	mockConfig2 := mock.NewMockConfig(mockCtl)
	p2pConfig := models.P2PConfig{
		Host:    "",
		Port:    9546,
		KeyPath: "/Users/yijiaren/Workspaces/git/chain5j/chain5j/conf/certs/ssl2/server/server-key.pem",
		// KeyPath:          "./testdata/node2/nodekey",
		CertPath:         "/Users/yijiaren/Workspaces/git/chain5j/chain5j/conf/certs/ssl2/server/server.pem",
		IsTls:            true,
		EnablePermission: false,
		MaxPeers:         MaxPeers,
		// StaticNodes:      []string{"/ip4/127.0.0.1/tcp/9545/p2p/QmVy5JASWLUns3Wwe91arP93rPPumkEQ7fQZ4GfpevxKbd"},
		StaticNodes: []string{"/ip4/127.0.0.1/tcp/9545/p2p/QmS3bGuEww8gziBHqt2Z6NpUNk4g4xj2nrBQ6s53FNWQve"},
		CaRoots: []string{
			"/Users/yijiaren/Workspaces/git/chain5j/chain5j/conf/certs/ssl/ca/ca.pem",
			"/Users/yijiaren/Workspaces/git/chain5j/chain5j/conf/certs/ssl2/ca/ca.pem",
			"/Users/yijiaren/Workspaces/git/chain5j/chain5j/conf/certs/ssl2/ca/ca-sub.pem",
			// "/Users/yijiaren/Workspaces/git/chain5j/chain5j/conf/certs/ssl2/server/server-all.pem",
		},
		Metrics:      true,
		MetricsLevel: 2,
	}
	mockConfig2.EXPECT().P2PConfig().Return(p2pConfig).AnyTimes()

	service2, err := NewP2P(
		context.Background(),
		WithConfig(mockConfig2),
	)
	if err != nil {
		t.Fatal(err)
	}

	go testPeerEvent(service2, &wg, t)

	err = service2.Start()
	if err != nil {
		t.Fatal(err)
	}

	// 接收消息
	go testMsgRecv(t, service2, &wg)

	// 等待连接
	time.Sleep(5 * time.Second)
	if len(p2pConfig.StaticNodes) > 0 {
		for _, node := range p2pConfig.StaticNodes {
			muAddr, err := multiaddr.NewMultiaddr(node)
			if err != nil {
				t.Error(err)
				continue
			}
			info, err := peer.AddrInfoFromP2pAddr(muAddr)
			if err != nil {
				t.Error(err)
				continue
			}
			go sendMsg(service2, models.P2PID(info.ID.Pretty()), &wg)
		}
	}

	wg.Add(1)
	wg.Wait()
}

func sendMsg(server protocol.P2PService, peerId models.P2PID, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	count := 0
	for {
		select {
		case <-ticker.C:
			count++
			server.Send(peerId, &models.P2PMessage{
				Type: 1,
				Peer: "",
				Data: []byte("I am " + server.Id().String() + "==>" + convutil.ToString(count)),
			})
		}
	}
}

func testMsgRecv(t *testing.T, server protocol.P2PService, wg *sync.WaitGroup) {
	recv := make(chan *models.P2PMessage, 10)
	sub := server.SubscribeMsg(1, recv)
	defer sub.Unsubscribe()

	for {
		select {
		case rmsg := <-recv:
			t.Log("recv p2p message", "msgType", rmsg.Type, "peerId", rmsg.Peer.String(), "msg", string(rmsg.Data))
		case err := <-sub.Err():
			t.Fatal(err)
		}
	}
}

func testPeerEvent(server protocol.P2PService, wg *sync.WaitGroup, t *testing.T) {
	wg.Add(1)
	defer wg.Done()

	newPeerCh := make(chan models.P2PID)
	sub := server.SubscribeNewPeer(newPeerCh)

	dropPeerch := make(chan models.P2PID)
	sub2 := server.SubscribeDropPeer(dropPeerch)

	for {
		select {
		case id := <-newPeerCh:
			t.Log("test new peer connect", "peer", id)
		case id := <-dropPeerch:
			t.Log("test peer disconnect", "peer", id)
			return
		case <-sub.Err():
			return
		case <-sub2.Err():
			return
		}
	}
}
