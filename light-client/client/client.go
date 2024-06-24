package client

import (
	"context"
	"strings"
	"time"

	"github.com/CESSProject/cess-dcdn-components/downloader"
	"github.com/CESSProject/cess-dcdn-components/p2p"
	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-go-tools/scheduler"
	"github.com/CESSProject/p2p-go/core"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
)

const (
	MAX_FLUSH_TIMES       = 512
	DEFAULT_NODE_NUM      = 128
	DEFAULT_TASK_INTERVAL = time.Minute * 5
)

type Client struct {
	*chain.ChainClient
	Mnemonic string
	cachers  scheduler.Selector
	storages scheduler.Selector
	*core.PeerNode
	signature.KeyringPair
	tmpDir      string
	uploadQueue chan UploadStats
}

type UploadStats struct {
	FileHash    string
	FilePath    string
	CallBack    func(UploadStats)
	SegmentInfo []chain.SegmentDataInfo
	Group       []UploadGroup
}

type UploadGroup struct {
	MinerID   string
	PeerID    string
	Saved     bool
	OnChain   bool
	FlushTime time.Time
}

type FileBox struct {
	FileHash    string      `json:"file_hash"`
	TotalSize   int64       `json:"total_size"`
	PaddingSize int64       `json:"padding_size"`
	Plates      []FilePlate `json:"file_plates"`
}

type FilePlate struct {
	SegmentHash string   `json:"segment_hash"`
	Files       []string `json:"file_names"`
	Indexs      []int    `json:"file_indexs"`
}

func NewClient(chainCli chain.ChainClient, peerNode *core.PeerNode, cachers, storages scheduler.Selector, mnemonic, tmpDir string) error {
	return nil
}

func (c *Client) RunDiscovery(ctx context.Context, bootNode string) error {
	ch := make(chan peer.AddrInfo, 256)

	go func() {
		count, cNum, sNum := 0, 0, 0
		for peer := range ch {
			if peer.ID.Size() == 0 {
				break
			}
			//filter

			resp, err := downloader.DailCacheNode(c.PeerNode, peer.ID)
			if err == nil && resp.Info != nil {
				//select neighbor cache node
				cNum++
				c.cachers.FlushPeerNodes(5*time.Second, peer)
			} else if strings.Contains(err.Error(), "") {
				c.storages.FlushPeerNodes(5*time.Second, peer)
				sNum++
			}
			count++
			if count > MAX_FLUSH_TIMES {
				count = 0
				time.Sleep(time.Minute * 5)
			}
			if cNum >= DEFAULT_NODE_NUM && sNum >= DEFAULT_NODE_NUM {
				time.Sleep(time.Second * 15)
			}
		}
	}()
	err := p2p.Subscribe(ctx, c.GetHost(), bootNode, ch)
	if err != nil {
		return errors.Wrap(err, "run discovery service error")
	}
	return nil
}
