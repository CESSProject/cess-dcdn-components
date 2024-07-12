package protocol

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/CESSProject/cess-dcdn-components/config"
	"github.com/CESSProject/cess-dcdn-components/protocol/contract"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
)

const (
	PERM_UNTRUST = 3
	PERM_COMMON  = 2
	PERM_TRUST   = 1
	PERM_OWNER   = 0

	POINT_LIMIT = 64
)

type CreditItem struct {
	Point         int
	Perm          int
	ReceivedBytes uint64 // received bytes, or equivalent by valid payment
	SentBytes     uint64 // sent bytes
	Checkout      int
	LastAccess    time.Time
}

type CreditMap struct {
	lock      *sync.Mutex
	cmap      map[string]CreditItem
	ethClient *contract.Client
}

func NewCreditManager(cli *contract.Client) *CreditMap {
	return &CreditMap{
		lock:      &sync.Mutex{},
		cmap:      make(map[string]CreditItem),
		ethClient: cli,
	}
}

func (c CreditMap) GetClient() *contract.Client {
	return c.ethClient
}

func (c *CreditMap) SetUserCredit(key string, cdelta int) {
	c.lock.Lock()
	defer c.lock.Unlock()
	credit := c.cmap[key]
	credit.Point += cdelta
	if cdelta > 0 {
		credit.ReceivedBytes = uint64(cdelta) * config.CACHE_BLOCK_SIZE
	} else {
		credit.SentBytes = uint64(-cdelta) * config.CACHE_BLOCK_SIZE
	}
	perm := 1 + credit.SentBytes/credit.ReceivedBytes
	if perm == 1 && credit.ReceivedBytes >= 1024*config.CACHE_BLOCK_SIZE {
		credit.Perm = PERM_TRUST
	}
	if perm >= 3 {
		credit.Perm = PERM_UNTRUST
	}
	credit.LastAccess = time.Now()
	c.cmap[key] = credit
}

func (c *CreditMap) SetCacherOwner(acc string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	credit := c.cmap[acc]
	credit.LastAccess = time.Now()
	credit.Perm = PERM_OWNER
	c.cmap[acc] = credit
}

func (c *CreditMap) GetUserCredit(key string) CreditItem {
	var (
		credit CreditItem
		ok     bool
	)
	c.lock.Lock()
	defer c.lock.Unlock()
	credit, ok = c.cmap[key]
	if !ok {
		credit.LastAccess = time.Now()
		c.cmap[key] = credit
	}
	return credit
}

func CheckPointLimit(item CreditItem, limit int, size uint64, isCacher bool) (int, bool) {
	var res bool
	point := (item.ReceivedBytes / size) / uint64(item.Checkout)
	if point > POINT_LIMIT {
		point = POINT_LIMIT
	}
	deadLine := limit + (PERM_UNTRUST-item.Perm)*int(point)
	if !isCacher {
		if item.Point <= 0 && float64(deadLine)-math.Abs(float64(item.Point)) <= 0 {
			res = false
		} else {
			res = true
		}
	} else {
		if item.Point > 0 && deadLine-item.Point <= 0 {
			res = false
		} else {
			res = true
		}
	}
	return deadLine, res
}

func (c *CreditMap) PaymentCacheCreditBill(key, peerId string, price, size uint64, creditLimit int, maxFee int64) ([]byte, []byte, error) {
	credit := c.GetUserCredit(key)
	bill, ok := CheckPointLimit(credit, creditLimit, size, true)
	if ok {
		return nil, nil, nil
	}

	bPeerId, err := base58.Decode(peerId)
	if err != nil {
		return nil, nil, err
	}
	info, err := QueryRegisterInfo(c.ethClient, key)
	if err != nil {
		return nil, nil, err
	}
	if !bytes.Equal(info.PeerId, bPeerId) {
		return nil, nil, errors.New("peer ID does not match the registered one")
	}
	value := int64(price * uint64(bill))

	opts, err := c.ethClient.NewTransactionOption(context.Background(), big.NewInt(value).String())
	if err != nil {
		return nil, nil, err
	}
	if value > maxFee {
		return nil, nil, fmt.Errorf("cacher order bill(%d) exceeds the preset maximum amount(%d)", value, maxFee)
	}
	_, orderId, err := CreateCacheOrder(c.ethClient, key, opts)
	if err != nil {
		return nil, nil, err
	}
	sign, err := c.ethClient.GetSignature(crypto.Keccak256Hash(orderId[:]).Bytes())
	if err != nil {
		return nil, nil, err
	}
	c.SetUserCredit(key, -bill)
	return orderId[:], sign, nil
}

func (c *CreditMap) CheckCredit(key string, data, sign []byte) bool {
	var (
		ok   bool
		res  bool
		item CreditItem
	)
	c.lock.Lock()
	defer c.lock.Unlock()
	item, ok = c.cmap[key]
	if !ok {
		item.LastAccess = time.Now()
		c.cmap[key] = item
	}
	_, res = CheckPointLimit(item, config.GetConfig().FreeDownloads, config.CACHE_BLOCK_SIZE, false)
	if len(data) > 0 && len(sign) > 0 {

		if c.ethClient.VerifySign(crypto.Keccak256Hash(data).Bytes(), sign) {
			return res
		}
		order, err := QueryCacheOrder(c.ethClient, [32]byte(data))
		if err != nil {
			return res
		}
		if order.Node != c.ethClient.Account {
			return res
		}
		term, err := QueryCurrencyTerm(c.ethClient)
		if err != nil {
			return res
		}
		if term.Cmp(order.Term.Add(order.Term, big.NewInt(4))) == 1 {
			return res
		}
		//claim Order
		price := big.NewInt(int64(config.GetConfig().CachePrice))
		num := order.Value.Div(order.Value, price).Int64()
		item.ReceivedBytes += uint64(num) * config.CACHE_BLOCK_SIZE
		item.Point += int(num)
		item.Checkout += 1
		item.LastAccess = time.Now()
		c.cmap[key] = item
		_, res = CheckPointLimit(item, config.GetConfig().FreeDownloads, config.CACHE_BLOCK_SIZE, false)
	}
	return res
}
