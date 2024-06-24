package config

import (
	"os"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	PROFILE_TEMPLATE = `# The rpc endpoint of the chain node
Rpc:
  # test network
  - "wss://testnet-rpc0.cess.cloud/ws/"
  - "wss://testnet-rpc1.cess.cloud/ws/"
  - "wss://testnet-rpc2.cess.cloud/ws/"
# The storage network you want to connect to, "mainnet" or "testnet"
Network: 
# Bootstrap Nodes
Boot:
  # test network
  - "_dnsaddr.boot-bucket-testnet.cess.cloud"
# CESS account mnemonic, it be used to query info on chain or send transactions
Mnemonic: "..."
# Service workspace
Workspace: "/"
# P2P communication port
P2P_Port: 4001
`
	CACHE_PROFILE_TEMPLATE = `
# User Files Cacher config
# File cache size, default 512G, (unit is byte)
CacheSize:
# File cache expiration time, default 3*60 minutes (unit is minutes)
Expiration:
# Directory to store file cache, default path: Workspace/filecache/
CacheDir:
# Price of downloading a single fragment (unit is 1/10^18 CESS or TCESS, is an integer)
CachePrice: 
`

	DEFAULT_CONFIG_PATH  = "./config.yaml"
	TESTNET_PROTO_PREFIX = "/testnet"
	MAINNET_PROTO_PREFIX = "/mainnet"
	DEVNET_PROTO_PREFIX  = "/devnet"
	DEFAULT_CACHE_PATH   = "./cache/"
	DEFAULT_WORKSPACE    = "./workspace/"
	DEFAULT_CACHE_PRICE  = 10000000000000000
)

var config Config

type Config struct {
	CachePrice uint64   `name:"CachePrice" toml:"CachePrice" yaml:"CachePrice"`
	CacheDir   string   `name:"CacheDir" toml:"CacheDir" yaml:"CacheDir"`
	Expiration int64    `name:"Expiration" toml:"Expiration" yaml:"Expiration"`
	CacheSize  int64    `name:"CacheSize" toml:"CacheSize" yaml:"CacheSize"`
	Mnemonic   string   `name:"Mnemonic" toml:"Mnemonic" yaml:"Mnemonic"`
	Rpc        []string `name:"Rpc" toml:"Rpc" yaml:"Rpc"`
	Boot       []string `name:"Boot" toml:"Boot" yaml:"Boot"`
	P2PPort    int      `name:"P2P_Port" toml:"P2P_Port" yaml:"P2P_Port"`
	WorkSpace  string   `name:"WorkSpace" toml:"WorkSpace" yaml:"WorkSpace"`
	Network    string   `name:"Network" toml:"Network" yaml:"Network"`
	KeyPair    signature.KeyringPair
}

func GetConfig() Config {
	return config
}

func ParseConfig(fpath string) error {
	if fpath == "" {
		fpath = DEFAULT_CONFIG_PATH
	}
	viper.SetConfigFile(fpath)
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		return errors.Wrap(err, "parse config file error")
	}
	err = viper.Unmarshal(&config)
	if err != nil {
		return errors.Wrap(err, "parse config file error")
	}

	if config.CacheSize <= 32*1024*1024*1024 {
		config.CacheSize = 32 * 1024 * 1024 * 1024
	}
	if config.Expiration <= 0 || config.Expiration > 7*24*60 {
		config.Expiration = 3 * 60
	}
	switch config.Network {
	case MAINNET_PROTO_PREFIX, TESTNET_PROTO_PREFIX, DEVNET_PROTO_PREFIX:
	default:
		config.Network = TESTNET_PROTO_PREFIX
	}
	if config.CacheDir == "" {
		config.CacheDir = DEFAULT_CACHE_PATH
	}
	if config.WorkSpace == "" {
		config.WorkSpace = DEFAULT_WORKSPACE
	}
	if config.CachePrice == 0 {
		config.CachePrice = DEFAULT_CACHE_PRICE
	}

	if _, err := os.Stat(config.CacheDir); err != nil {
		err = os.MkdirAll(config.CacheDir, 0755)
		if err != nil {
			return errors.Wrap(err, "parse config file error")
		}
	}
	if _, err := os.Stat(config.WorkSpace); err != nil {
		err = os.MkdirAll(config.WorkSpace, 0755)
		if err != nil {
			return errors.Wrap(err, "parse config file error")
		}
	}
	keyPair, err := signature.KeyringPairFromSecret(config.Mnemonic, 0)
	if err != nil {
		return errors.Wrap(err, "parse config file error")
	}
	config.KeyPair = keyPair
	return nil
}
