package config

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	DEFAULT_CONFIG_PATH    = "./config.yaml"
	TESTNET_PROTO_PREFIX   = "/testnet"
	MAINNET_PROTO_PREFIX   = "/mainnet"
	DEVNET_PROTO_PREFIX    = "/devnet"
	DEFAULT_CACHE_PATH     = "./cache/"
	DEFAULT_WORKSPACE      = "./workspace/"
	DEFAULT_CACHE_PRICE    = "10000000000000000"
	DEFAULT_GAS_FREE_CAP   = 108694000460
	DEFAULT_GAS_LIMIT      = 30000000
	DEFAULT_DOWNLOAD_POINT = 4
	CACHE_BLOCK_SIZE       = 8 * 1024 * 1024
)

var (
	_default DefaultConfig
)

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
}

type DefaultConfig struct {
	CessChainConfig
	P2PConfig
	CacherConfig
	CdnProtoConfig
}

type CessChainConfig struct {
	Rpc      []string
	Network  string
	Mnemonic string
}

type P2PConfig struct {
	Boots     []string
	P2PPort   int
	WorkSpace string
}

type CacherConfig struct {
	CachePrice    string
	CacheDir      string
	Expiration    int64
	CacheSize     int64
	FreeDownloads int
}

type CdnProtoConfig struct {
	ChainId           int64
	Staking           string
	GasFreeCap        int64
	GasLimit          uint64
	ContractAddresss  map[string]string
	NodeTokenId       string
	NodeAccPrivateKey string
	TokenAccAddress   string
	TokenAccSign      string
}

func GetConfig() DefaultConfig {
	return _default
}

func ParseDefaultConfig(fpath string) error {

	err := ParseCommonConfig(fpath, "yaml", &_default)
	if err != nil {
		return err
	}

	if _default.CacheSize <= 32*1024*1024*1024 {
		_default.CacheSize = 32 * 1024 * 1024 * 1024
	}
	if _default.Expiration <= 0 || _default.Expiration > 7*24*60 {
		_default.Expiration = 3 * 60
	}
	// switch _default.Network {
	// case MAINNET_PROTO_PREFIX, TESTNET_PROTO_PREFIX, DEVNET_PROTO_PREFIX:
	// default:
	// 	_default.Network = TESTNET_PROTO_PREFIX
	// }

	if _default.CacheDir == "" {
		_default.CacheDir = DEFAULT_CACHE_PATH
	}
	if _default.WorkSpace == "" {
		_default.WorkSpace = DEFAULT_WORKSPACE
	}
	if _default.CachePrice == "" {
		_default.CachePrice = DEFAULT_CACHE_PRICE
	}
	if _default.FreeDownloads == 0 {
		_default.FreeDownloads = DEFAULT_DOWNLOAD_POINT
	}

	if _, err := os.Stat(_default.CacheDir); err != nil {
		err = os.MkdirAll(_default.CacheDir, 0755)
		if err != nil {
			return errors.Wrap(err, "parse default config file error")
		}
	}
	if _, err := os.Stat(_default.WorkSpace); err != nil {
		err = os.MkdirAll(_default.WorkSpace, 0755)
		if err != nil {
			return errors.Wrap(err, "parse default config file error")
		}
	}
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
