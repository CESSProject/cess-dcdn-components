package test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/CESSProject/cess-dcdn-components/cd2n-client/client"
	cdnlib "github.com/CESSProject/cess-dcdn-components/cd2n-lib"
	"github.com/CESSProject/cess-dcdn-components/cd2n-lib/credit"
	"github.com/CESSProject/cess-dcdn-components/cd2n-lib/p2p"
	"github.com/CESSProject/cess-dcdn-components/cd2n-lib/protocol"
	"github.com/CESSProject/cess-dcdn-components/cd2n-lib/types"
	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-tools/scheduler"
	p2pgo "github.com/CESSProject/p2p-go"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/out"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58/base58"
)

var (
	TestPeers = []string{
		"12D3KooWLK8DUHynHLxiPpoDtmEKh85Hs95V7mGNfUAZk4muFwJV",
		"12D3KooWCzusyToYzcURTpnnNb1Yop7LNdW6a19gwUcWpfD6Bhx5",
		"12D3KooWDf5K8XF6TW1XsL7hT9aj3A4zMfWtJnPpQp1eJ8KqB1RB",
		"12D3KooWEdgkwwXMiNgbNzFzbouPDm6bJ1NSKP8KktThN8BVzMyU",
		"12D3KooWKhis2Mzyq1E5LoHeWNwgUrit8TdfQz1TmMnrQxV8YDyW",
	}
	rpcs = []string{
		"wss://testnet-rpc.cess.cloud/ws",
		"wss://testnet-rpc.cess.network/ws",
	}
)

func TestDailCacheNode(t *testing.T) {
	peerNode := NewTestP2pNode()
	if peerNode == nil {
		t.Fatal("new test p2p node error")
	}
	defer peerNode.Close()

	time.Sleep(15 * time.Second)
	peerId, err := ConvertToPeerId(TestPeers[0])
	if err != nil {
		t.Fatal("convert peerId error", err)
	}
	// err = peerNode.ReadFileAction(peerId, "ajhdfasdfa", "fasdfasdf", "./file", 8*1024*1024)
	// if err != nil {
	// 	t.Fatal("test read file action error", err)
	// }
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

func TestQueryCacheNodes(t *testing.T) {
	cli := InitTestClient()
	if cli == nil {
		t.Fatal("init test client error")
	}
	for {
		t.Log("peers: ", cli.CacheCli.Peerstore().Peers().Len())
		//cli.PeerNode.GetDHTable().FindProviders()
		bPeerId, err := base58.Decode("12D3KooWRWnGtKeMT7AZchjfkavhS3uWKo4tDgc8T2nUaswNHCGN") //12D3KooWLK8DUHynHLxiPpoDtmEKh85Hs95V7mGNfUAZk4muFwJV
		if err != nil {
			t.Log("register cache node error", err)
			continue
		}
		t.Log("request 12D3KooWRWnGtKeMT7AZchjfkavhS3uWKo4tDgc8T2nUaswNHCGN")
		resp, err := cdnlib.QueryFileInfoFromCache(
			cli.CacheCli.Host, peer.ID(bPeerId),
			types.CacheRequest{
				AccountId: cli.GetPublickey(),
				WantUrl:   "https://gw.crust-gateway.xyz/ipfs/bafybeicf5cnabenlocs35n56eggjryq277dxstnkpnv6h5lqkidfd7nsqu",
				// WantFile: path.Join(
				// 	"8e962f1d4c5567f942c94596448acb3c1194485f4afeee584eb48978340b32cd",
				// 	"153cecaef9be527209c9be556b89b03c9411b42cfd9875e744f37fbad4997676",
				// ),
			})
		if err != nil {
			t.Log("query cache node file info error", err)
			time.Sleep(time.Second * 10)
			continue
		}
		jbytes, err := json.Marshal(resp)
		if err != nil {
			t.Log("marshal response error", err)
			continue
		}
		t.Log("query success", string(jbytes))
		time.Sleep(5 * time.Second)
	}
}

func TestGetFileCacheNodes(t *testing.T) {
	cli := InitTestClient()
	if cli == nil {
		t.Fatal("init test client error")
	}
	for {
		t.Log("peers: ", cli.CacheCli.Peerstore().Peers().Len())
		//cli.PeerNode.GetDHTable().FindProviders()
		bPeerId, err := base58.Decode("12D3KooWRWnGtKeMT7AZchjfkavhS3uWKo4tDgc8T2nUaswNHCGN") //12D3KooWLK8DUHynHLxiPpoDtmEKh85Hs95V7mGNfUAZk4muFwJV
		if err != nil {
			t.Log("register cache node error", err)
			continue
		}
		t.Log("request 12D3KooWRWnGtKeMT7AZchjfkavhS3uWKo4tDgc8T2nUaswNHCGN")
		err = cdnlib.DownloadFileFromCache(
			cli.CacheCli.Host, peer.ID(bPeerId), "./test.mp4",
			types.CacheRequest{
				AccountId: cli.GetPublickey(),
				WantUrl:   "https://gw.crust-gateway.xyz/ipfs/bafybeicf5cnabenlocs35n56eggjryq277dxstnkpnv6h5lqkidfd7nsqu",
			},
		)
		if err != nil {
			t.Log("query cache node file info error", err)
			time.Sleep(time.Second * 10)
			continue
		}
		t.Log("query success")
		break
	}

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
	rec := make(chan<- peer.AddrInfo, 12)
	go p2p.StartDiscoveryFromDHT(context.Background(), peerNode.GetHost(), peerNode.GetDHTable(), p2p.DISCOVERY_RENDEZVOUS, time.Second, rec)
	go p2p.StartDiscoveryFromMDNS(context.Background(), peerNode.GetHost(), rec)
	return peerNode
}

func ConvertToPeerId(id string) (peer.ID, error) {
	return peer.Decode(id)
}

func InitTestClient() *client.Client {
	chainSdk, err := cess.New(
		context.Background(),
		cess.ConnectRpcAddrs(rpcs),
		cess.Mnemonic("skill income exile ethics sick excess sea deliver medal junk update fault"),
		cess.TransactionTimeout(time.Second*30),
	)
	if err != nil {
		log.Println("init cess chain client error", err)
		return nil
	}

	for {
		syncSt, err := chainSdk.SystemSyncState()
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}
		if syncSt.CurrentBlock == syncSt.HighestBlock {
			out.Tip(fmt.Sprintf("Synchronization main chain completed: %d", syncSt.CurrentBlock))
			break
		}
		out.Tip(fmt.Sprintf("In the synchronization main chain: %d ...", syncSt.CurrentBlock))
		time.Sleep(time.Second * time.Duration(Ternary(int64(syncSt.HighestBlock-syncSt.CurrentBlock)*6, 30)))
	}
	storageNode, err := p2pgo.New(
		context.Background(),
		p2pgo.ListenPort(4001),
		p2pgo.Workspace("./testP2P"),
		p2pgo.BootPeers([]string{"_dnsaddr.boot-miner-testnet.cess.network"}),
		p2pgo.ProtocolPrefix(""),
	)
	if err != nil {
		log.Println("init storage p2p client error", err)
		return nil
	}

	key, err := p2p.Identification(path.Join("./testP2P", ".key"))
	if err != nil {
		log.Println("init p2p identification error", err)
		return nil
	}
	cacheNode, err := p2p.NewPeerNode(
		"/cdnnet", p2p.Version, []string{""},
		libp2p.Identity(key),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 4002)),
	)
	if err != nil {
		log.Println("init P2P Node error", err)
		return nil
	}
	cacherSelc, err := scheduler.NewNodeSelector(
		scheduler.PRIORITY_STRATEGY,
		"./cacher_list",
		64,
		int64(time.Millisecond*1000),
		int64(time.Hour*6),
	)
	if err != nil {
		log.Println("init node selector error", err)
		return nil
	}
	storageSelc, err := scheduler.NewNodeSelector(
		scheduler.PRIORITY_STRATEGY,
		"./storage_list",
		128,
		int64(time.Millisecond*1000),
		int64(time.Hour*6),
	)
	if err != nil {
		log.Println("init node selector error", err)
		return nil
	}
	cli, err := protocol.NewClient(
		protocol.AccountPrivateKey("c126731601a2b8e1a10149b548972e3dc577b9b2e174dcb5de326d4d6e928df5"),
		protocol.ChainID(11330),
		protocol.ConnectionRpcAddresss(rpcs),
		protocol.EthereumGas(108694000460, 30000000),
	)
	if err != nil {
		log.Println("init ethereum client error", err)
		return nil
	}
	contract, err := protocol.NewProtoContract(
		cli.GetEthClient(),
		"0x7352188979857675C3aD1AA6662326ebD6DDBf6d",
		cli.Account.Hex(),
		cli.NewTransactionOption,
		cli.SubscribeFilterLogs,
	)
	if err != nil {
		log.Println("init contract client error", err)
		return nil
	}
	cmg, err := credit.NewCreditManager(
		cli.Account.Hex(),
		"test_credit_record",
		"10000000000",
		0, contract,
	)
	if err != nil {
		log.Println("init credit client error", err)
		return nil
	}
	sk := cli.GetPrivateKey()
	lightClient := client.NewClient(*chainSdk, storageNode, cacheNode, cacherSelc, storageSelc, cmg, &sk)
	lightClient.SetConfig("", "./temp", "1000000000000000000000", 100)
	go lightClient.RunDiscovery(context.Background(), "_dnsaddr.boot-miner-testnet.cess.network")
	return lightClient
}

func Ternary(a, b int64) int64 {
	if a > b {
		return b
	}
	return a
}
