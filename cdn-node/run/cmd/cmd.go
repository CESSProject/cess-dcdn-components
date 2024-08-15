package cmd

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CESSProject/cess-dcdn-components/cdn-lib/types"
	"github.com/CESSProject/cess-dcdn-components/cdn-node/cache"
	"github.com/CESSProject/cess-dcdn-components/config"
	"github.com/CESSProject/cess-dcdn-components/logger"
	"github.com/CESSProject/cess-dcdn-components/protocol"
	"github.com/CESSProject/cess-dcdn-components/protocol/contract"
	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-tools/cacher"
	"github.com/CESSProject/cess-go-tools/scheduler"
	p2pgo "github.com/CESSProject/p2p-go"
	"github.com/CESSProject/p2p-go/out"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   types.CACHE_NAME,
	Short: "light cache service based on CESS network",
}

func Execute() {
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func InitCmd() {
	rootCmd.AddCommand(
		cmd_run(),
		cmd_exit_network(),
	)
	rootCmd.PersistentFlags().StringP("config", "c", "", "custom profile")
}

func cmd_run() *cobra.Command {
	return &cobra.Command{
		Use:                   "run",
		Short:                 "Running services",
		DisableFlagsInUseLine: true,
		Run:                   cmd_run_func,
	}
}

func cmd_exit_network() *cobra.Command {
	return &cobra.Command{
		Use:                   "exit",
		Short:                 "exit node from CESS CDN network and redeem staking",
		DisableFlagsInUseLine: true,
		Run:                   cmd_exit_func,
	}
}

func cmd_exit_func(cmd *cobra.Command, args []string) {
	cpath, _ := cmd.Flags().GetString("config")
	if cpath == "" {
		cpath, _ = cmd.Flags().GetString("c")
		if cpath == "" {
			logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("empty config file path")
			log.Println("empty config file path")
			return
		}
	}

	if err := config.ParseDefaultConfig(cpath); err != nil {
		log.Println("error", err)
		return
	}
	conf := config.GetConfig()
	cli, err := contract.NewClient(
		contract.AccountPrivateKey(conf.NodeAccPrivateKey),
		contract.ChainID(conf.ChainId),
		contract.ConnectionRpcAddresss(conf.Rpc),
		contract.EthereumGas(conf.GasFreeCap, conf.GasLimit),
	)
	if err != nil {
		log.Fatal(err)
	}
	opts, err := cli.NewTransactionOption(context.Background(), "")
	if err != nil {
		log.Fatal(err)
	}
	err = protocol.ExitNetwork(cli, opts)
	if err != nil {
		log.Fatal(err)
	}
}

func cmd_run_func(cmd *cobra.Command, args []string) {
	cpath, _ := cmd.Flags().GetString("config")
	if cpath == "" {
		cpath, _ = cmd.Flags().GetString("c")
		if cpath == "" {
			logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("empty config file path")
			log.Println("empty config file path")
			return
		}
	}

	if err := config.ParseDefaultConfig(cpath); err != nil {
		logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("error", err)
		log.Println("error", err)
		return
	}

	ctx := context.Background()
	conf := config.GetConfig()
	chainSdk, err := cess.New(
		ctx,
		cess.ConnectRpcAddrs(conf.Rpc),
		cess.Mnemonic(conf.Mnemonic),
		cess.TransactionTimeout(time.Second*30),
	)
	if err != nil {
		logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("init cess chain client error", err)
		log.Println("init cess chain client error", err)
		return
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
	log.Println("node p2p port", conf.P2PPort)
	peerNode, err := p2pgo.New(
		ctx,
		p2pgo.ListenPort(conf.P2PPort),
		p2pgo.Workspace(conf.WorkSpace),
		p2pgo.BootPeers(conf.Boots),
		p2pgo.ProtocolPrefix(conf.Network),
	)

	if err != nil {
		logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("init P2P Node error", err)
		log.Println("init P2P Node error", err)
		return
	}
	defer peerNode.Close()

	if _, err = os.Stat(types.TempDir); err != nil {
		if err = os.Mkdir(types.TempDir, 0755); err != nil {
			logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("make temp dir error ", err)
			log.Println("make temp dir error", err)
			return
		}
	}

	cacheModule := cacher.NewCacher(
		time.Duration(conf.Expiration)*time.Minute,
		conf.CacheSize,
		conf.CacheDir,
	)
	selector, err := scheduler.NewNodeSelector(
		scheduler.PRIORITY_STRATEGY,
		"./node_list",
		cache.MaxNeigborNum,
		int64(time.Millisecond*300),
		int64(time.Hour*6),
	)
	if err != nil {
		logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("init node selector error", err)
		log.Println("init node selector error", err)
		return
	}
	cli, err := contract.NewClient(
		contract.AccountPrivateKey(conf.NodeAccPrivateKey),
		contract.ChainID(conf.ChainId),
		contract.ConnectionRpcAddresss(conf.Rpc),
		contract.EthereumGas(conf.GasFreeCap, conf.GasLimit),
	)
	if err != nil {
		logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("init ethereum client error", err)
		log.Println("init ethereum client error", err)
		return
	}
	cli.AddWorkContract(conf.ContractAddresss)

	if info, err := protocol.QueryRegisterInfo(cli, cli.Account.Hex()); err != nil {
		// tokenId, err := hex.DecodeString(conf.NodeTokenId)
		// if err != nil {
		// 	log.Println("decode token Id error", err)
		// 	return
		// }
		sign, err := hex.DecodeString(conf.TokenAccSign)
		if err != nil {
			logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("decode sign error", err)
			log.Println("decode sign error", err)
			return
		}
		opts, err := cli.NewTransactionOption(context.Background(), conf.Staking)
		if err != nil {
			logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("new transaction option error", err)
			log.Println("new transaction option error", err)
			return
		}
		err = protocol.RegisterNode(
			cli, conf.TokenAccAddress,
			peerNode.GetHost().ID().String(),
			conf.NodeTokenId, sign, opts,
		)
		if err != nil {
			logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("register node on chain error", err)
			log.Println("register node on chain error", err)
			return
		}
	} else {
		logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Info("get registered node information:", info.String())
		log.Println("get registered node information:", info.String())
	}

	cacher := cache.NewCacher(chainSdk, cacheModule, peerNode, selector, cli)
	cacher.SetConfig(conf.CachePrice, cli.Account.Bytes())
	cacher.RestoreCacheFiles(conf.CacheDir)
	//peerNode.EnableRecv()

	log.Println("cache node network profix", peerNode.ProtocolPrefix)
	logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Info("🚀 CESS CDN cache service is running ...")
	log.Println("🚀 CESS CDN cache service is running ...")

	ctx, stop := context.WithCancel(ctx)
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		<-signals
		log.Println("get system cancel signal.")
		stop()
		log.Println("wait for service to stop ...")
	}()
	go func() {
		if len(conf.Boots) <= 0 {
			stop()
			logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("empty boot node list.")
			log.Println("please configure boot node.")
			return
		}
		err = cacher.RunDiscovery(ctx, conf.Boots[0])
		if err != nil {
			stop()
			logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("run discovery service error", err)
			log.Println("run discovery service error", err)
			return
		}
	}()
	go func() {
		opts, err := cli.NewTransactionOption(context.Background(), "")
		if err != nil {
			logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("new transacton option error", err)
			log.Println("new transacton option error", err)
			stop()
			return
		}
		err = protocol.ClaimWorkRewardServer(context.Background(), cli, opts)
		if err != nil {
			logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Error("run claim work reward server error", err)
			log.Println("run claim work reward server error", err)
			stop()
			return
		}
	}()
	cacher.RunDownloadServer(ctx, 0)
	logger.GetGlobalLogger().GetLogger(types.LOG_NODE).Info("🔚 CESS CDN cache service done.")
	log.Println("🔚 CESS CDN cache service done.")
}

func Ternary(a, b int64) int64 {
	if a > b {
		return b
	}
	return a
}
