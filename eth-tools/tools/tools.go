package tools

import (
	"context"
	"encoding/hex"
	"errors"
	"log"
	"math/big"
	"strings"

	"github.com/CESSProject/cess-dcdn-components/protocol/contract"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type Config struct {
	ChainId         int64    `name:"ChainId" toml:"ChainId" yaml:"ChainId"`
	ChainRpcs       []string `name:"ChainRpc" toml:"ChainRpc" yaml:"ChainRpc"`
	GasFreeCap      int64    `name:"GasFreeCap" toml:"GasFreeCap" yaml:"GasFreeCap"`
	GasLimit        uint64   `name:"GasLimit" toml:"GasLimit" yaml:"GasLimit"`
	ContractAddress string   `name:"ContractAddress" toml:"ContractAddress" yaml:"ContractAddress"`
}

func Keccak256HashWithContractPrefix(hash common.Hash) common.Hash {
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	return crypto.Keccak256Hash(
		prefix,
		hash.Bytes(),
	)
}

func BuyCacherToken(c Config, sk, value string) (string, error) {
	client, err := contract.NewClient(
		contract.AccountPrivateKey(sk),
		contract.ChainID(c.ChainId),
		contract.ConnectionRpcAddresss(c.ChainRpcs),
		contract.EthereumGas(c.GasFreeCap, c.GasLimit),
	)
	if err != nil {
		return "", err
	}
	opts, err := client.NewTransactionOption(
		context.Background(), value,
	)
	if err != nil {
		return "", err
	}
	tokenCli, err := contract.NewToken(
		common.HexToAddress(c.ContractAddress), client.GetEthClient(),
	)
	if err != nil {
		return "", err
	}
	tx, err := tokenCli.MintToken(opts, client.Account)
	log.Println("transaction hash:", tx.Hash())
	if err != nil {
		return "", err
	}
	var tokenId string
	if err = client.SubscribeFilterLogs(
		context.Background(), ethereum.FilterQuery{
			Addresses: []common.Address{
				common.HexToAddress(c.ContractAddress),
			},
		}, func(l types.Log) bool {
			event, err := tokenCli.ParseMintToken(l)
			if err != nil {
				event, err := tokenCli.ParseTransfer(l)
				if err == nil && event.To == client.Account {
					tokenId = event.TokenId.String()
					return false
				}
				return true
			} else if event.Owner == client.Account {
				tokenId = event.TokenId.String()
				return false
			}
			return true
		},
	); err != nil {
		return "", err
	}
	return tokenId, nil
}

func CacherAuthorizationSign(tokenAccSk, nodeAcc, tokenId string) ([]byte, error) {
	sk, err := crypto.HexToECDSA(tokenAccSk)
	if err != nil {
		return nil, err
	}
	tid, _ := big.NewInt(0).SetString(tokenId, 10)
	byteAcc, err := hex.DecodeString(strings.TrimPrefix(nodeAcc, "0x"))
	if err != nil {
		return nil, err
	}
	hash := Keccak256HashWithContractPrefix(
		crypto.Keccak256Hash(
			byteAcc,
			tid.Bytes(),
		),
	)
	sign, err := crypto.Sign(hash.Bytes(), sk)
	if err != nil {
		return nil, err
	}
	if len(sign) != 65 {
		return nil, errors.New("invalid signature length")
	}
	sign[64] += 27
	return sign, nil
}
