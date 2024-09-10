package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/CESSProject/cess-dcdn-components/cd2n-client/client"
	"github.com/CESSProject/cess-dcdn-components/cd2n-lib/credit"
	"github.com/CESSProject/cess-dcdn-components/cd2n-lib/p2p"
	"github.com/CESSProject/cess-dcdn-components/cd2n-lib/protocol"
	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-tools/scheduler"
	p2pgo "github.com/CESSProject/p2p-go"
	"github.com/CESSProject/p2p-go/out"
	"github.com/libp2p/go-libp2p"
)

var rpcs = []string{
	"wss://testnet-rpc.cess.cloud/ws",
	"wss://testnet-rpc.cess.network/ws",
}

func main() {
	// cli := InitTestClient()
	// if cli == nil {
	// 	log.Fatal("init test client error")
	// }
	// if len(os.Args) < 2 {
	// 	log.Fatal("Please enter the resource you want to download")
	// }
	// for {
	// 	bPeerId, err := base58.Decode("12D3KooWQgrQLZ3x9WC43z9wFoJ6oZvTUVEqbZfkdF1kywfVinVp") //12D3KooWLK8DUHynHLxiPpoDtmEKh85Hs95V7mGNfUAZk4muFwJV
	// 	if err != nil {
	// 		log.Println("register cache node error", err)
	// 		continue
	// 	}
	// 	log.Println("request 12D3KooWQgrQLZ3x9WC43z9wFoJ6oZvTUVEqbZfkdF1kywfVinVp")
	// 	resp, err := cdnlib.QueryFileInfoFromCache(
	// 		cli.CacheCli.Host, peer.ID(bPeerId),
	// 		types.CacheRequest{
	// 			AccountId: cli.GetPublickey(),
	// 			WantUrl:   os.Args[1],
	// 			// WantFile: path.Join(
	// 			// 	"8e962f1d4c5567f942c94596448acb3c1194485f4afeee584eb48978340b32cd",
	// 			// 	"153cecaef9be527209c9be556b89b03c9411b42cfd9875e744f37fbad4997676",
	// 			// ),
	// 		})
	// 	if err != nil {
	// 		log.Println("query cache node file info error", err)
	// 		time.Sleep(time.Second * 10)
	// 		continue
	// 	}
	// 	if resp.Status != "hit" {
	// 		log.Println("wait CDepin node to cache resource")
	// 		time.Sleep(time.Second * 5)
	// 		continue
	// 	}
	// 	log.Println("CDepin node cached resource success.")
	// 	break
	// }
	// for {
	// 	//cli.PeerNode.GetDHTable().FindProviders()
	// 	bPeerId, err := base58.Decode("12D3KooWQgrQLZ3x9WC43z9wFoJ6oZvTUVEqbZfkdF1kywfVinVp") //12D3KooWLK8DUHynHLxiPpoDtmEKh85Hs95V7mGNfUAZk4muFwJV
	// 	if err != nil {
	// 		log.Println("register cache node error", err)
	// 		continue
	// 	}

	// 	err = cdnlib.DownloadFileFromCache(
	// 		cli.CacheCli.Host, peer.ID(bPeerId), "./test.mp4",
	// 		types.CacheRequest{
	// 			AccountId: cli.GetPublickey(),
	// 			WantUrl:   os.Args[1],
	// 		},
	// 	)
	// 	if err != nil {
	// 		log.Println("query cache node file info error", err)
	// 		time.Sleep(time.Second * 10)
	// 		continue
	// 	}
	// 	log.Println("get resource success")
	// 	break
	// }
	SetupHttpServer()
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
