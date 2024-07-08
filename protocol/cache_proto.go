package protocol

import (
	"context"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/CESSProject/cess-dcdn-components/contract"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

type NodeInfo struct {
	Created   bool
	Collerate *big.Int
	TokenId   *big.Int
	PeerId    []byte
}

type Order struct {
	Value   *big.Int
	Creater common.Address
	Node    common.Address
	Term    *big.Int
}

func QueryRegisterInfo(cli *contract.Client, nodeAcc string) (NodeInfo, error) {
	var info NodeInfo
	protoContract, err := contract.NewCacheProto(
		cli.GetContractAddress(contract.DEFAULT_CACHE_PROTO_CONTRACT_NAME),
		cli.GetEthClient(),
	)
	if err != nil {
		return info, errors.Wrap(err, "query node register info error")
	}
	info, err = protoContract.Node(&bind.CallOpts{}, common.HexToAddress(nodeAcc))
	if err != nil {
		return info, errors.Wrap(err, "query node register info error")
	}
	return info, nil
}

func QueryCacheOrder(cli *contract.Client, orderId [32]byte) (Order, error) {
	var order Order
	protoContract, err := contract.NewCacheProto(
		cli.GetContractAddress(contract.DEFAULT_CACHE_PROTO_CONTRACT_NAME),
		cli.GetEthClient(),
	)
	if err != nil {
		return order, errors.Wrap(err, "query cache order info error")
	}
	order, err = protoContract.Order(&bind.CallOpts{}, orderId)
	if err != nil {
		return order, errors.Wrap(err, "query cache order info error")
	}
	return order, nil
}

func QueryCurrencyTerm(cli *contract.Client) (*big.Int, error) {
	protoContract, err := contract.NewCacheProto(
		cli.GetContractAddress(contract.DEFAULT_CACHE_PROTO_CONTRACT_NAME),
		cli.GetEthClient(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "query currency term error")
	}
	term, err := protoContract.GetCurrencyTerm(&bind.CallOpts{})
	if err != nil {
		return nil, errors.Wrap(err, "query currency term error")
	}
	return term, nil
}

func CreateCacheOrder(cli *contract.Client, nodeAddress string, opts *bind.TransactOpts) (string, [32]byte, error) {

	var orderId []byte
	contractAddr := cli.GetContractAddress(contract.DEFAULT_CACHE_PROTO_CONTRACT_NAME)
	protoContract, err := contract.NewCacheProto(contractAddr, cli.GetEthClient())
	if err != nil {
		return "", [32]byte{}, errors.Wrap(err, "create and payment cache order error")
	}
	nodeAcc := common.HexToAddress(nodeAddress)
	tx, err := protoContract.CacheOrderPayment(opts, nodeAcc)
	if err != nil {
		return "", [32]byte{}, errors.Wrap(err, "create and payment cache order error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	if err = cli.SubscribeFilterLogs(
		ctx,
		ethereum.FilterQuery{
			Addresses: []common.Address{contractAddr},
		},
		func(l types.Log) bool {
			event, err := protoContract.ParseOrderPayment(l)
			if err != nil {
				return true
			}
			if event.NodeAcc == nodeAcc {
				orderId = event.OrderId[:]
				return false
			}
			return true
		},
	); err != nil {
		return tx.Hash().Hex(), [32]byte{}, errors.Wrap(err, "create and payment cache order error")
	}
	if orderId == nil {
		err = errors.New("order ID not obtained")
		return tx.Hash().Hex(), [32]byte{}, errors.Wrap(err, "create and payment cache order error")
	}
	return tx.Hash().Hex(), [32]byte(orderId), nil
}

func RegisterNode(cli *contract.Client, tokenAccAddr, tokenId, sign, peerId string, opts *bind.TransactOpts) error {

	contractAddr := cli.GetContractAddress(contract.DEFAULT_CACHE_PROTO_CONTRACT_NAME)
	protoContract, err := contract.NewCacheProto(contractAddr, cli.GetEthClient())
	if err != nil {
		return errors.Wrap(err, "register cache node error")
	}
	tokenAcc := common.HexToAddress(tokenAccAddr)
	bigId, ok := big.NewInt(0).SetString(tokenId, 10)
	if !ok {
		err = errors.New("bad token Id")
		return errors.Wrap(err, "register cache node error")
	}
	bPeerId, err := hex.DecodeString(peerId)
	if err != nil {
		return errors.Wrap(err, "register cache node error")
	}
	bSign, err := hex.DecodeString(sign)
	if err != nil {
		return errors.Wrap(err, "register cache node error")
	}
	_, err = protoContract.Staking(opts, cli.Account, tokenAcc, bigId, bPeerId, bSign)
	if err != nil {
		return errors.Wrap(err, "register cache node error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	if err = cli.SubscribeFilterLogs(
		ctx,
		ethereum.FilterQuery{
			Addresses: []common.Address{contractAddr},
		},
		func(l types.Log) bool {
			event, err := protoContract.ParseStaking(l)
			if err != nil {
				return true
			}
			if event.NodeAcc == cli.Account {
				return false
			}
			return true
		},
	); err != nil {
		return errors.Wrap(err, "register cache node error")
	}
	return nil
}

func ClaimWorkReward(cli *contract.Client, opts *bind.TransactOpts) (string, error) {

	var reward string = "0"
	contractAddr := cli.GetContractAddress(contract.DEFAULT_CACHE_PROTO_CONTRACT_NAME)
	protoContract, err := contract.NewCacheProto(contractAddr, cli.GetEthClient())
	if err != nil {
		return reward, errors.Wrap(err, "claim work reward error")
	}
	_, err = protoContract.Claim(opts)
	if err != nil {
		return reward, errors.Wrap(err, "claim work reward error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	if err = cli.SubscribeFilterLogs(
		ctx,
		ethereum.FilterQuery{
			Addresses: []common.Address{contractAddr},
		},
		func(l types.Log) bool {
			event, err := protoContract.ParseClaim(l)
			if err != nil {
				return true
			}
			if event.NodeAcc == cli.Account {
				reward = event.Reward.String()
				return false
			}
			return true
		},
	); err != nil {
		return reward, errors.Wrap(err, "claim work reward error")
	}
	return reward, nil
}

func ExitNetwork(cli *contract.Client, opts *bind.TransactOpts) error {
	contractAddr := cli.GetContractAddress(contract.DEFAULT_CACHE_PROTO_CONTRACT_NAME)
	protoContract, err := contract.NewCacheProto(contractAddr, cli.GetEthClient())
	if err != nil {
		return errors.Wrap(err, "node exit network error")
	}
	_, err = protoContract.Exit(opts)
	if err != nil {
		return errors.Wrap(err, "node exit network error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	if err = cli.SubscribeFilterLogs(
		ctx,
		ethereum.FilterQuery{
			Addresses: []common.Address{contractAddr},
		},
		func(l types.Log) bool {
			event, err := protoContract.ParseClaim(l)
			if err != nil {
				return true
			}
			if event.NodeAcc == cli.Account {
				return false
			}
			return true
		},
	); err != nil {
		return errors.Wrap(err, "node exit network error")
	}
	return nil
}
