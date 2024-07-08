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
# The number of free downloads allowed for users
FreeDownloads:
`
	CONTRACT_PROFILE_TEMPLATE = `
# Cache protocol config
# The Id of chain where the EVM contract is deployed
ChainId:
# Ethereum gas free cap (default is 108694000460)
GasFreeCap:
#Ethereum gas limit (default is 30000000)
GasLimit:
#Cache protocol smart contracts for node registration, order and revenue management, etc.
ContractAddress:
# NFT Token Id that needs to be bound to the cache node(license for cache node to join network)
NodeTokenId:
# Cache Node's ethereum account private key (Hexadecimal string, not including '0x')
NodeAccPrivateKey:
Token owner's ethereum account address (Hexadecimal string, including '0x')
TokenAccAddress:
The signature of the token owner's account on the message pair, obtained through the 'eth-tools'
TokenAccSign:
`

	DEFAULT_CONFIG_PATH    = "./config.yaml"
	TESTNET_PROTO_PREFIX   = "/testnet"
	MAINNET_PROTO_PREFIX   = "/mainnet"
	DEVNET_PROTO_PREFIX    = "/devnet"
	DEFAULT_CACHE_PATH     = "./cache/"
	DEFAULT_WORKSPACE      = "./workspace/"
	DEFAULT_CACHE_PRICE    = 10000000000000000
	DEFAULT_GAS_FREE_CAP   = 108694000460
	DEFAULT_GAS_LIMIT      = 30000000
	DEFAULT_DOWNLOAD_POINT = 4
	CACHE_BLOCK_SIZE       = 8 * 1024 * 1024
)

var config Config

type Config struct {
	CachePrice        uint64   `name:"CachePrice" toml:"CachePrice" yaml:"CachePrice"`
	CacheDir          string   `name:"CacheDir" toml:"CacheDir" yaml:"CacheDir"`
	Expiration        int64    `name:"Expiration" toml:"Expiration" yaml:"Expiration"`
	CacheSize         int64    `name:"CacheSize" toml:"CacheSize" yaml:"CacheSize"`
	Mnemonic          string   `name:"Mnemonic" toml:"Mnemonic" yaml:"Mnemonic"`
	Rpc               []string `name:"Rpc" toml:"Rpc" yaml:"Rpc"`
	Boot              []string `name:"Boot" toml:"Boot" yaml:"Boot"`
	P2PPort           int      `name:"P2P_Port" toml:"P2P_Port" yaml:"P2P_Port"`
	WorkSpace         string   `name:"WorkSpace" toml:"WorkSpace" yaml:"WorkSpace"`
	Network           string   `name:"Network" toml:"Network" yaml:"Network"`
	ChainId           int64    `name:"ChainId" toml:"ChainId" yaml:"ChainId"`
	GasFreeCap        int64    `name:"GasFreeCap" toml:"GasFreeCap" yaml:"GasFreeCap"`
	GasLimit          uint64   `name:"GasLimit" toml:"GasLimit" yaml:"GasLimit"`
	ContractAddress   string   `name:"ContractAddress" toml:"ContractAddress" yaml:"ContractAddress"`
	NodeTokenId       string   `name:"NodeTokenId" toml:"NodeTokenId" yaml:"NodeTokenId"`
	NodeAccPrivateKey string   `name:"NodeAccPrivateKey" toml:"NodeAccPrivateKey" yaml:"NodeAccPrivateKey"`
	TokenAccAddress   string   `name:"TokenAccAddress" toml:"TokenAccAddress" yaml:"TokenAccAddress"`
	TokenAccSign      string   `name:"TokenAccSign" toml:"TokenAccSign" yaml:"TokenAccSign"`
	FreeDownloads     int      `name:"FreeDownloads" toml:"FreeDownloads" yaml:"FreeDownloads"`
	KeyPair           signature.KeyringPair
}

func GetConfig() Config {
	return config
}

func ParseConfig(fpath string) error {

	err := ParseCommonConfig(fpath, "yaml", &config)
	if err != nil {
		return err
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
	if config.FreeDownloads == 0 {
		config.FreeDownloads = DEFAULT_DOWNLOAD_POINT
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

func ParseCommonConfig(fpath, ctype string, config interface{}) error {
	if fpath == "" {
		fpath = DEFAULT_CONFIG_PATH
	}
	viper.SetConfigFile(fpath)
	viper.SetConfigType(ctype)
	err := viper.ReadInConfig()
	if err != nil {
		return errors.Wrap(err, "parse config file error")
	}
	err = viper.Unmarshal(config)
	if err != nil {
		return errors.Wrap(err, "parse config file error")
	}
	return nil
}
