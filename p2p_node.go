// Package p2p
//
// @author: xwc1125
package p2p

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/chain5j/chain5j-p2p/p2pgmtls"
	"github.com/chain5j/chain5j-p2p/p2ptls"
	"github.com/chain5j/chain5j-pkg/event"
	"github.com/chain5j/chain5j-protocol/models"
	"github.com/chain5j/chain5j-protocol/protocol"
	"github.com/chain5j/logger"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	ci "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	noise "github.com/libp2p/go-libp2p-noise"
	p2pTls "github.com/libp2p/go-libp2p-tls"
	"github.com/multiformats/go-multiaddr"
	"github.com/tjfoc/gmsm/gmtls"
)

// p2pNode 节点信息
type p2pNode struct {
	log    logger.Logger
	ctx    context.Context
	cancel context.CancelFunc

	config protocol.Config

	host      host.Host    // 当前节点host
	peerId    peer.ID      // 当前节点的PeerID
	p2pPrvKey ci.PrivKey   // p2p的私钥
	cert      *Certificate // 证书（如果是tls）

	notify        *p2pNotify
	streamManager *p2pStreamManager
	p2pService    *p2pService
	peerCount     int32 // 已经连接的节点数量

	newPeerFeed   event.Feed              // 新节点事件
	peerDropFeed  event.Feed              // 节点断开连接事件
	handshakeFeed event.Feed              // 需要handshake事件
	scope         event.SubscriptionScope // scope管理

	feeds *sync.Map // uint-->*subscriber
}

// newP2PNode 创建新的p2p节点
func newP2PNode(rootCtx context.Context, config protocol.Config) (*p2pNode, error) {
	log := logger.New("p2p")
	ctx, cancel := context.WithCancel(rootCtx)
	logging.SetAllLoggers(logging.LevelError)

	prvKey, cert, err := getPrvKey(log, config)
	if err != nil {
		return nil, err
	}
	// 通过私钥获取peerId
	peerId, err := peer.IDFromPrivateKey(prvKey)
	if err != nil {
		return nil, err
	}

	node := &p2pNode{
		log:    log,
		ctx:    ctx,
		cancel: cancel,

		config: config,

		peerId:    peerId,
		p2pPrvKey: prvKey,
		cert:      cert,

		feeds: new(sync.Map),
	}

	node.notify = &p2pNotify{
		log,
		node,
	}

	if node.streamManager, err = newStreamManager(ctx, config, node); err != nil {
		log.Error("new stream manager err", "err", err)
		return nil, err
	}

	return node, nil
}

// start 启动p2p节点
func (n *p2pNode) start() (err error) {
	n.host, err = makeHost(n.log, n.config, n.p2pPrvKey, n.cert)
	if err != nil {
		return err
	}
	// 设置数据流处理方法
	n.host.SetStreamHandler(ProtocolID, n.handleStream)
	n.host.Network().Notify(n.notify)

	// todo 将共识中的节点也加入到可信任的节点中
	// 添加静态节点
	for _, dst := range n.config.P2PConfig().StaticNodes {
		n.addPeer(dst)
	}

	for {
		n.connectToPeers()
		select {
		case <-n.ctx.Done():
			return nil
		case <-time.After(1 * time.Minute):
			break
		}
	}
}

func (n *p2pNode) stop() {
	n.cancel()
	n.streamManager.stop()
	if n.host != nil {
		n.host.Close()
	}
}

// setServer 设置p2p service
func (n *p2pNode) setP2PService(p2pService *p2pService) {
	n.p2pService = p2pService
}

func (n *p2pNode) id() peer.ID {
	return n.peerId
}

// addPeer 添加目标peer
func (n *p2pNode) addPeer(destUrl string) error {
	mAddr, err := multiaddr.NewMultiaddr(destUrl)
	if err != nil {
		n.log.Error("new multi addr err", "addr", destUrl, "err", err)
		return err
	}

	// 展开地址信息
	info, err := peer.AddrInfoFromP2pAddr(mAddr)
	if err != nil {
		n.log.Error("addr to addrInfo err", "addr", destUrl, "err", err)
		return err
	}
	if n.isMetrics(2) {
		n.log.Debug("add peer ok", "peerId", info.ID.Pretty())
	}
	// 在peerstore中添加目标的对等多地址。这将在libp2p创建连接和流时使用
	n.peerStore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	// 权限处理
	if n.config.P2PConfig().EnablePermission {
		err := n.peerStore().Put(info.ID, "authorize", true)
		if err != nil {
			n.log.Error("add authorize key error", "peerId", info.ID.Pretty(), "error", err)
		}
	}

	if len(n.host.Network().ConnsToPeer(info.ID)) == 0 &&
		!strings.EqualFold(n.peerId.Pretty(), info.ID.Pretty()) {
		go func() {
			if err := n.host.Connect(n.ctx, *info); err != nil {
				n.log.Error("connect peer", "peerId", info.ID, "err", err)
			} else {
				// 和目标peer创建一个新的流
				s, err := n.host.NewStream(n.ctx, info.ID, ProtocolID)
				if err != nil {
					n.log.Error("open stream error", "peerId", info.ID, "err", err)
					return
				}
				n.streamManager.addStream(s)
			}
		}()
	}

	return nil
}

// dropPeer 移除peerId
func (n *p2pNode) dropPeer(id peer.ID) {
	n.peerStore().ClearAddrs(id)
	n.host.Network().ClosePeer(id)
}

func (n *p2pNode) existPeer(id peer.ID) bool {
	if _, ok := n.streamManager.streams.Load(id); ok {
		return true
	}
	return false
}

func (n *p2pNode) peerStore() peerstore.Peerstore {
	return n.host.Peerstore()
}

// isAuthorizationNode 判断peerId是否授权
func (n *p2pNode) isAuthorizationNode(id peer.ID) bool {
	// 未开启授权，允许所有节点通过
	if !n.config.P2PConfig().EnablePermission {
		return true
	}

	v, err := n.peerStore().Get(id, "authorize")
	if err != nil {
		return false
	}

	return v.(bool)
}

func (n *p2pNode) peers() map[string]*models.P2PInfo {
	infos := make(map[string]*models.P2PInfo)
	for _, id := range n.peerStore().PeersWithAddrs() {
		infos[id.Pretty()] = &models.P2PInfo{
			Id: models.P2PID(id.Pretty()),
		}

		conn := n.host.Network().ConnsToPeer(id)
		if len(conn) > 0 {
			infos[id.Pretty()].NetUrl = fmt.Sprintf("%s/p2p/%s", conn[0].RemoteMultiaddr().String(), id.Pretty())
			infos[id.Pretty()].Connected = true
			continue
		}

		if len(n.peerStore().PeerInfo(id).Addrs) > 0 {
			infos[id.Pretty()].NetUrl = fmt.Sprintf("%s/p2p/%s", n.peerStore().PeerInfo(id).Addrs[0].String(), id.Pretty())
		}
	}

	return infos
}

// connectToPeers 连接到peers
func (n *p2pNode) connectToPeers() error {
	var connected int
	for _, id := range n.peerStore().PeersWithAddrs() {
		if n.id() == id {
			if n.isMetrics(3) {
				n.log.Trace("connect to self", "peerId", id)
			}
			continue
		}
		if connected > MaxPeers {
			break
		}
		peerInfo := n.host.Peerstore().PeerInfo(id)
		// 连接到peerId
		if len(n.host.Network().ConnsToPeer(id)) > 0 {
			if n.isMetrics(3) {
				n.log.Debug("already connect Peer", "peerId", id)
			}
			connected++
			continue
		}

		n.log.Info("connect to peer", "id", id.Pretty())
		if err := n.host.Connect(n.ctx, peerInfo); err != nil {
			n.log.Error("connect peer error", "peerId", id, "err", err)
		} else {
			s, err := n.host.NewStream(n.ctx, id, ProtocolID)
			if err != nil {
				n.log.Error("open stream error", "peerId", id, "err", err)
				continue
			}
			n.streamManager.addStream(s)
		}
	}

	return nil
}

func (n *p2pNode) connectedPeers() []models.P2PID {
	var ids []models.P2PID

	// TODO 需要缓存，不从libp2p获取
	for _, id := range n.peerStore().PeersWithAddrs() {
		if len(n.host.Network().ConnsToPeer(id)) > 0 {
			ids = append(ids, models.P2PID(id.Pretty()))
		}
	}

	return ids
}

// handleStream 处理数据流
func (n *p2pNode) handleStream(s network.Stream) {
	if n.isMetrics(3) {
		n.log.Debug("got a new p2p stream", "protocolId", string(s.Protocol()))
	}

	n.streamManager.addStream(s)
}

// sendMsg 发送消息
func (n *p2pNode) sendMsg(peerId peer.ID, msg *models.P2PMessage) error {
	return n.streamManager.send(msg, peerId)
}

// makeHost 创建host
func makeHost(log logger.Logger, config protocol.Config, prvKey ci.PrivKey, cert *Certificate) (host.Host, error) {
	addr := getP2PAddr(config)
	muAddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		log.Error("new multi addr err", "addr", addr, "err", err)
		return nil, err
	}

	// 创建p2p 网关
	connGater := newConnGater(config)

	opts := []libp2p.Option{
		libp2p.UserAgent("chain5j"),
		libp2p.DefaultTransports,          // 默认使用tcp
		libp2p.ListenAddrs(muAddr),        // 本地节点监听地址
		libp2p.ConnectionGater(connGater), // 添加连接的gate
		libp2p.DisableRelay(),             // 配置libp2p以禁用中继传输
		libp2p.ConnectionManager(connmgr.NewConnManager(100, 400, time.Minute)), // 连接管理器来防止对等方拥有过多的连接
	}

	// 是否启动tls
	if !config.P2PConfig().IsTls {
		// opts = append(opts, libp2p.Identity(prvKey)) // 私钥
		// opts = append(opts, libp2p.DefaultSecurity)  // 安全的P2P[常规]
		// 默认的tls
		opts = append(opts, libp2p.Identity(prvKey))
		opts = append(opts, libp2p.Security(p2pTls.ID, p2pTls.New))
		opts = append(opts, libp2p.Security(noise.ID, noise.New))
	} else {
		// TODO 【xwc1125】使用证书获取私钥
		// 根据私钥判断
		if cert != nil {
			switch cert.TlsType {
			case 20:
				// 国密
				log.Info("the type of the prvKey found is sm2. use gm tls security.")
				caRoots := make([][]byte, 0)
				if config.P2PConfig().CaRoots != nil {
					for _, root := range config.P2PConfig().CaRoots {
						rootBytes, err := ioutil.ReadFile(root)
						if err != nil {
							return nil, err
						}
						caRoots = append(caRoots, rootBytes)
					}
				}
				transport, err := p2pgmtls.New(cert.Certificate.(*gmtls.Certificate), caRoots)
				if err != nil {
					return nil, err
				}
				opts = append(opts, libp2p.Identity(prvKey))
				opts = append(opts, libp2p.Security(p2pgmtls.ProtocolID, transport))
			default:
				caRoots := make([][]byte, 0)
				if config.P2PConfig().CaRoots != nil {
					for _, root := range config.P2PConfig().CaRoots {
						rootBytes, err := ioutil.ReadFile(root)
						if err != nil {
							return nil, err
						}
						caRoots = append(caRoots, rootBytes)
					}
				}
				transport, err := p2ptls.New(cert.Certificate.(*tls.Certificate), caRoots)
				if err != nil {
					return nil, err
				}
				opts = append(opts, libp2p.Identity(prvKey))
				opts = append(opts, libp2p.Security(p2ptls.ProtocolID, transport))
			}
		}
	}

	// 使用p2p创建host
	basicHost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	log.Info("local p2p node", "peerId", basicHost.ID().Pretty())
	return basicHost, nil
}

// getP2PAddr 获取p2p的addr，如/ip4/0.0.0.0/tcp/9545
func getP2PAddr(config protocol.Config) string {
	var addr string
	if len(config.P2PConfig().Host) > 0 {
		address := net.ParseIP(config.P2PConfig().Host)
		if address != nil {
			// 没有匹配上
			addr = fmt.Sprintf("/ip4/%s/tcp/%d", config.P2PConfig().Host, config.P2PConfig().Port)
		}
	}
	if len(addr) == 0 {
		addr = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", config.P2PConfig().Port)
	}
	return addr
}

func (n *p2pNode) handshakeSuccess(peerId peer.ID) {
	// TODO 链接数量统计， 用于最大节点数量统计
	if n.isMetrics(1) {
		n.log.Info("new peer handshake success", "peerId", peerId.Pretty())
	}

	if n.p2pService != nil {
		n.newPeerFeed.Send(models.P2PID(peerId.Pretty()))
	}
}

func (n *p2pNode) isMetrics(metrics uint64) bool {
	return n.config.P2PConfig().IsMetrics(metrics)
}
