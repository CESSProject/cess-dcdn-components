package client

import (
	"context"
	"encoding/json"
	"log"
	"time"

	cdnlib "github.com/CESSProject/cess-dcdn-components/cdn-lib"
	"github.com/CESSProject/cess-dcdn-components/cdn-lib/types"
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
	EthClient *contract.Client
	Mnemonic  string
	Cachers   scheduler.Selector
	Storages  scheduler.Selector
	*core.PeerNode
	CacheFeeLimit string
	tmpDir        string
	cmp           *protocol.CreditMap
	osQueue       chan types.CacheResponse //order settlement queue
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
		EthClient:   cli,
		PeerNode:    peerNode,
		Cachers:     cachers,
		Storages:    storages,
		cmp:         protocol.NewCreditManager(cli),
	}
}

func (c *Client) SetConfig(Mnemonic, tmpDir string, CacheFeeLimit string, uploadQueueSize int) {
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
			c.PeerNode, peer.ID(bpid),
			&cdnlib.Options{
				Account: c.EthClient.Account.Bytes(),
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
			case peer := <-peerCh:
				if peer.ID.Size() == 0 {
					break
				}
				c.Storages.FlushPeerNodes(5*time.Second, peer)
			case peer := <-cacheCh:
				if peer.ID.Size() == 0 {
					break
				}
				log.Println("find cacher ", peer.ID)
				//
				if c.Cachers.GetPeersNumber() >= MaxCacheNodeNum {
					continue
				}
				if res, err := cdnlib.DailCacheNode(c.PeerNode, peer.ID); err != nil {
					log.Println("dail", peer.ID, " error ", err)
					continue
				} else {
					jbytes, _ := json.Marshal(res)
					log.Println("dail cacher success, response: ", string(jbytes))
				}
				c.Cachers.FlushPeerNodes(15*time.Second, peer)
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
			p2p.DISCOVERY_RENDEZVOUS,
			time.Second*3, cacheCh,
		)
	}()
	err = p2p.Subscribe(ctx, c.GetHost(), c.GetBootnode(), time.Minute, peerCh)
	if err != nil {
		return errors.Wrap(err, "run discovery service error")
	}
	return nil
}
