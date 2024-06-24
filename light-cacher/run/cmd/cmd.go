package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CESSProject/cess-dcdn-components/config"
	"github.com/CESSProject/cess-dcdn-components/light-cacher/cache"
	"github.com/CESSProject/cess-dcdn-components/light-cacher/ctype"
	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-tools/cacher"
	"github.com/CESSProject/cess-go-tools/scheduler"
	p2pgo "github.com/CESSProject/p2p-go"
	"github.com/CESSProject/p2p-go/out"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   ctype.CACHE_NAME,
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
		cmd_config(),
		cmd_run(),
	)
	rootCmd.PersistentFlags().StringP("config", "c", "", "custom profile")
}

func cmd_config() *cobra.Command {
	return &cobra.Command{
		Use:                   "config",
		Short:                 "Generate configuration file",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			f, err := os.Create(config.DEFAULT_CONFIG_PATH)
			if err != nil {
				log.Fatal("[err]", err)
			}
			defer f.Close()
			_, err = f.WriteString(config.PROFILE_TEMPLATE)
			if err != nil {
				log.Fatal("[err]", err)
			}
			_, err = f.WriteString(config.CACHE_PROFILE_TEMPLATE)
			if err != nil {
				log.Fatal("[err]", err)
			}
			f.Sync()
			log.Println("[ok] generate a config file in", config.DEFAULT_CONFIG_PATH)
		},
	}
}

func cmd_run() *cobra.Command {
	return &cobra.Command{
		Use:                   "run",
		Short:                 "Running services",
		DisableFlagsInUseLine: true,
		Run:                   cmd_run_func,
	}
}

func cmd_run_func(cmd *cobra.Command, args []string) {
	cpath, _ := cmd.Flags().GetString("config")
	if cpath == "" {
		cpath, _ = cmd.Flags().GetString("c")
		if cpath == "" {
			log.Println("empty config file path")
			return
		}
	}

	if err := config.ParseConfig(cpath); err != nil {
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

	peerNode, err := p2pgo.New(
		ctx,
		p2pgo.ListenPort(conf.P2PPort),
		p2pgo.Workspace(conf.WorkSpace),
		p2pgo.BootPeers(conf.Boot),
		p2pgo.ProtocolPrefix(conf.Network),
	)

	if err != nil {
		log.Println("init cess chain client error", err)
		return
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
		log.Println("init cess chain client error", err)
		return
	}
	cacher := cache.NewCacher(chainSdk, cacheModule, peerNode, selector)

	cacher.SetConfig(conf.KeyPair, conf.CachePrice)
	cacher.RestoreCacheFiles(conf.CacheDir)
	cacher.RegisterP2pTsFileServiceHandle(cacher.ReadCacheService)

	log.Println("cess light cacher service is running ...")
	ctx, stop := context.WithCancel(ctx)
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		<-signals
		stop()
		log.Println("wait for service to stop ...")
	}()
	go func() {
		defer stop()
		if len(conf.Boot) <= 0 {
			log.Println("please configure boot node.")
			return
		}
		err = cacher.RunDiscovery(ctx, conf.Boot[0])
		if err != nil {
			log.Println(err)
			return
		}
	}()
	cacher.RunDownloadServer(ctx, 0)
	log.Println("cess light cacher service done.")
}

func Ternary(a, b int64) int64 {
	if a > b {
		return b
	}
	return a
}
