package client

import (
	"context"
	"log"
	"time"

	cdnlib "github.com/CESSProject/cess-dcdn-components/cdn-lib"
	"github.com/CESSProject/cess-dcdn-components/cdn-node/types"
	"github.com/CESSProject/cess-dcdn-components/config"
	"github.com/CESSProject/cess-dcdn-components/p2p"
	"github.com/CESSProject/cess-dcdn-components/protocol"
	"github.com/CESSProject/cess-dcdn-components/protocol/contract"
	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-go-tools/scheduler"
	"github.com/CESSProject/p2p-go/core"
	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
)

const (
	MAX_FLUSH_TIMES       = 512
	DEFAULT_NODE_NUM      = 128
	DEFAULT_TASK_INTERVAL = time.Minute * 5
)

var (
	MaxStorageNodeNum = 128
	MaxCacheNodeNum   = 128
)

type Client struct {
	*chain.ChainClient
	Mnemonic string
	cachers  scheduler.Selector
	storages scheduler.Selector
	*core.PeerNode
	CacheFeeLimit int64
	tmpDir        string
	cmp           *protocol.CreditMap
	osQueue       chan types.QueryResponse //order settlement queue
	uploadQueue   chan UploadStats
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

func NewClient(chainCli chain.ChainClient, peerNode *core.PeerNode, cachers, storages scheduler.Selector, cli *contract.Client) *Client {
	//mnemonic, tmpDir string
	return &Client{
		ChainClient: &chainCli,
		PeerNode:    peerNode,
		cachers:     cachers,
		storages:    storages,
		cmp:         protocol.NewCreditManager(cli),
	}
}

func (c *Client) SetConfig(Mnemonic, tmpDir string, CacheFeeLimit int64, uploadQueueSize int) {
	c.Mnemonic = Mnemonic
	c.tmpDir = tmpDir
	c.CacheFeeLimit = CacheFeeLimit
	c.uploadQueue = make(chan UploadStats, uploadQueueSize)
}

func (c *Client) PaymentCacheOrder() error {
	for resp := range c.osQueue {
		acc := common.BytesToAddress(resp.Info.Account).Hex()
		orderId, sign, err := c.cmp.PaymentCacheCreditBill(
			acc, resp.Info.PeerId, resp.Info.Price,
			config.CACHE_BLOCK_SIZE, resp.Info.CreditLimit, c.CacheFeeLimit,
		)
		if err != nil {
			log.Println("payment cache order error", err)
			continue
		}
		if orderId == nil || sign == nil {
			continue
		}
		bpid, err := base58.Decode(string(resp.Info.PeerId))
		if err != nil {
			log.Println("payment cache order error", err)
			continue
		}
		res, err := cdnlib.QueryFileInfoFromCache(
			c.PeerNode, peer.ID(bpid), "", "",
			&cdnlib.Options{
				Account: c.cmp.GetClient().Account.Bytes(),
				Data:    orderId,
				Sign:    sign,
			},
		)
		if err != nil {
			log.Println("payment cache order error", err)
			continue
		}
		if res.Status != types.STATUS_OK {
			log.Println("payment cache order error, response from cacher not is ok")
		}
	}
	return nil
}

func (c *Client) RunDiscovery(ctx context.Context, bootNode string) error {
	var err error
	peerCh := make(chan peer.AddrInfo, MaxStorageNodeNum)
	cacheCh := make(chan peer.AddrInfo, MaxCacheNodeNum)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case peer := <-cacheCh:
				if peer.ID.Size() == 0 {
					break
				}
				c.cachers.FlushPeerNodes(5*time.Second, peer)
			case peer := <-peerCh:
				if peer.ID.Size() == 0 {
					break
				}
				//
				if c.cachers.GetPeersNumber() >= MaxStorageNodeNum {
					continue
				}
				if _, err := cdnlib.DailCacheNode(c.PeerNode, peer.ID); err == nil {
					continue
				}
				c.storages.FlushPeerNodes(5*time.Second, peer)
			}

		}
	}()
	go func() {
		err = p2p.StartDiscoveryFromMDNS(ctx, c.GetHost(), cacheCh)
	}()
	go func() {
		err = p2p.StartDiscoveryFromDHT(
			ctx,
			c.GetHost(),
			c.GetDHTable(),
			c.GetRendezvousVersion(),
			time.Second*3, cacheCh,
		)
	}()
	err = p2p.Subscribe(ctx, c.GetHost(), c.GetBootnode(), time.Minute, peerCh)
	if err != nil {
		return errors.Wrap(err, "run discovery service error")
	}
	return nil
}
