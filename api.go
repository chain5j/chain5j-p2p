// Package p2p
//
// @author: xwc1125
package p2p

import (
	"fmt"

	"github.com/chain5j/chain5j-pkg/util/hexutil"
	"github.com/chain5j/chain5j-protocol/models"
	"github.com/chain5j/chain5j-protocol/protocol"
)

type API struct {
	p protocol.P2PService
}

func newAPI(p protocol.P2PService) *API {
	return &API{p: p}
}

func (a *API) Info() map[string]interface{} {
	result := make(map[string]interface{})

	result["id"] = a.p.Id()
	result["neturl"] = a.p.NetURL()

	return result
}

func (a *API) Peers() map[string]*models.P2PInfo {
	urls := a.p.P2PInfo()
	return urls
}

type AdminApi struct {
	service protocol.P2PService
}

func newP2PAdminApi(service protocol.P2PService) interface{} {
	return &AdminApi{
		service: service,
	}
}

// AddPeer 添加节点
func (api *AdminApi) AddPeer(url string) error {
	return api.service.AddPeer(url)
}

// DropPeer 删除节点
func (api *AdminApi) DropPeer(id string) error {
	return api.service.DropPeer(models.P2PID(id))
}

type NetAPI struct {
	net            protocol.P2PService
	networkVersion uint64
}

func newNetAPI(net protocol.P2PService, networkVersion uint64) *NetAPI {
	return &NetAPI{net, networkVersion}
}

func (s *NetAPI) Listening() bool {
	return true // always listening
}

func (s *NetAPI) PeerCount() hexutil.Uint {
	return hexutil.Uint(len(s.net.RemotePeers()))
}

func (s *NetAPI) Version() string {
	return fmt.Sprintf("%d", s.networkVersion)
}
