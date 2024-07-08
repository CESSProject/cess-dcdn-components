package protocol

import (
	"encoding/hex"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/CESSProject/cess-dcdn-components/config"
	"github.com/CESSProject/cess-dcdn-components/contract"
	"github.com/ethereum/go-ethereum/crypto"
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

func CheckPoint(item CreditItem, limit int, size uint64) (int, bool) {
	var res bool
	point := (item.ReceivedBytes / size) / uint64(item.Checkout)
	if point > POINT_LIMIT {
		point = POINT_LIMIT
	}
	deadLine := limit + (PERM_UNTRUST-item.Perm)*int(point)
	if (deadLine == 0 && item.Point == 0) || (item.Point < 0 &&
		math.Abs(float64(item.Point)-float64(deadLine)) > 0) {
		res = false
	} else {
		res = true
	}
	return deadLine, res
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
	_, res = CheckPoint(item, config.GetConfig().FreeDownloads, config.CACHE_BLOCK_SIZE)
	if len(data) > 0 && len(sign) > 0 {
		bkey, err := hex.DecodeString(key)
		if err != nil {
			return res
		}
		if !crypto.VerifySignature(
			bkey, crypto.Keccak256Hash(data).Bytes(), sign) {
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
		_, res = CheckPoint(item, config.GetConfig().FreeDownloads, config.CACHE_BLOCK_SIZE)
	}
	return res
}
