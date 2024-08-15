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

	"github.com/CESSProject/cess-dcdn-components/cdn-client/client"
	cdnlib "github.com/CESSProject/cess-dcdn-components/cdn-lib"
	"github.com/CESSProject/cess-dcdn-components/cdn-node/cache"
	"github.com/CESSProject/cess-dcdn-components/p2p"
	"github.com/CESSProject/cess-dcdn-components/protocol/contract"
	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-tools/scheduler"
	p2pgo "github.com/CESSProject/p2p-go"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/out"
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

func TestDailCacheNodes(t *testing.T) {
	cli := InitTestClient()
	if cli == nil {
		t.Fatal("init test client error")
	}
	for {
		t.Log("peers: ", cli.PeerNode.Peerstore().Peers().Len())
		//cli.PeerNode.GetDHTable().FindProviders()
		bPeerId, err := base58.Decode("12D3KooWLK8DUHynHLxiPpoDtmEKh85Hs95V7mGNfUAZk4muFwJV") //12D3KooWLK8DUHynHLxiPpoDtmEKh85Hs95V7mGNfUAZk4muFwJV
		if err != nil {
			t.Log("register cache node error", err)
			continue
		}
		resp, err := cdnlib.QueryFileInfoFromCache(
			cli.PeerNode, peer.ID(bPeerId),
			&cdnlib.Options{
				Account: cli.EthClient.Account[:],
				WantFile: path.Join(
					"db48efe9868085043c211635233583f38d0c6f25724d3e42d69cfb4a422708f5",
					"945e3bfe85a6e1ded2553691b4d6c2d62e7a50cf8511bf7a0c942f2d33e9d055",
				),
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
		// num := cli.Cachers.GetPeersNumber()
		// if num == 0 {
		// 	time.Sleep(time.Second * 3)
		// 	continue
		// }
		// log.Println("get peers number", num)
		// if num > cache.MaxNeigborNum {
		// 	num = cache.MaxNeigborNum
		// }
		// itoa, err := cli.Cachers.NewPeersIterator(num)
		// if err != nil {
		// 	t.Log("new peers iterator", err)
		// 	time.Sleep(time.Second * 5)
		// 	continue
		// }

		// for peer, ok := itoa.GetPeer(); ok; peer, ok = itoa.GetPeer() {
		// 	resp, err := cdnlib.QueryFileInfoFromCache(
		// 		cli.PeerNode, peer.ID,
		// 		"79f1363e7c6c027bf3ab601db3926e4cd84e22ed3be731387a3c50c8ad5d60b6",
		// 		"a6a9dd87f12f3121f3396af946e40d526edb7d02c0d848bb1244de6844cf5761",
		// 		&cdnlib.Options{
		// 			Account: cli.EthClient.Account[:],
		// 		})
		// 	if err != nil {
		// 		t.Log("query cache node file info error", err)
		// 		//cli.Cachers.Feedback(peer.ID.String(), false)
		// 		time.Sleep(time.Second * 10)
		// 		continue
		// 	}
		// 	jbytes, err := json.Marshal(resp)
		// 	if err != nil {
		// 		t.Log("marshal response error", err)
		// 		continue
		// 	}
		// 	t.Log("query success", string(jbytes))
		// }
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
	peerNode, err := p2pgo.New(
		context.Background(),
		p2pgo.ListenPort(4001),
		p2pgo.Workspace("./testP2P"),
		p2pgo.BootPeers([]string{"_dnsaddr.boot-miner-testnet.cess.network"}),
		p2pgo.ProtocolPrefix(""),
	)
	if err != nil {
		log.Println("init P2P Node error", err)
		return nil
	}
	cacherSelc, err := scheduler.NewNodeSelector(
		scheduler.PRIORITY_STRATEGY,
		"./cacher_list",
		cache.MaxNeigborNum,
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
		cache.MaxNeigborNum,
		int64(time.Millisecond*1000),
		int64(time.Hour*6),
	)
	if err != nil {
		log.Println("init node selector error", err)
		return nil
	}
	cli, err := contract.NewClient(
		contract.AccountPrivateKey("c126731601a2b8e1a10149b548972e3dc577b9b2e174dcb5de326d4d6e928df5"),
		contract.ChainID(11330),
		contract.ConnectionRpcAddresss(rpcs),
		contract.EthereumGas(108694000460, 30000000),
	)
	if err != nil {
		log.Println("init ethereum client error", err)
		return nil
	}
	lightClient := client.NewClient(*chainSdk, peerNode, cacherSelc, storageSelc, cli)
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
