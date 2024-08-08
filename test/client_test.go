package test

import (
	"context"
	"encoding/json"
	"log"
	"testing"
	"time"

	cdnlib "github.com/CESSProject/cess-dcdn-components/cdn-lib"
	p2pgo "github.com/CESSProject/p2p-go"
	"github.com/CESSProject/p2p-go/core"
	"github.com/libp2p/go-libp2p/core/peer"
)

var (
	TestPeers = []string{
		"12D3KooWCzusyToYzcURTpnnNb1Yop7LNdW6a19gwUcWpfD6Bhx5",
		"12D3KooWDf5K8XF6TW1XsL7hT9aj3A4zMfWtJnPpQp1eJ8KqB1RB",
		"12D3KooWEdgkwwXMiNgbNzFzbouPDm6bJ1NSKP8KktThN8BVzMyU",
		"12D3KooWKhis2Mzyq1E5LoHeWNwgUrit8TdfQz1TmMnrQxV8YDyW",
	}
)

func TestDailCacheNode(t *testing.T) {
	peerNode := NewTestP2pNode()
	if peerNode == nil {
		t.Fatal("new test p2p node error")
	}
	time.Sleep(15 * time.Second)
	peerId, err := ConvertToPeerId(TestPeers[2])
	if err != nil {
		t.Fatal("convert peerId error", err)
	}
	resp, err := cdnlib.DailCacheNode(peerNode, peerId)
	if err != nil {
		t.Fatal("dail cache node error", err)
	}
	jbytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatal("marshal response error", err)
	}
	t.Log("success", string(jbytes))
}

func NewTestP2pNode() *core.PeerNode {
	peerNode, err := p2pgo.New(
		context.Background(),
		p2pgo.ListenPort(4001),
		p2pgo.Workspace("./testP2P"),
		p2pgo.BootPeers([]string{"_dnsaddr.boot-miner-testnet.cess.network"}),
		p2pgo.ProtocolPrefix(""),
	)
	if err != nil {
		log.Println("init cess p2p node error", err)
		return nil
	}
	return peerNode
}

func ConvertToPeerId(id string) (peer.ID, error) {
	return peer.Decode(id)
}
