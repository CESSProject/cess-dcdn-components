package credit

import (
	"sync"
	"time"
)

const (
	PRIVILEGE_UNTRUST = 3
	PRIVILEGE_COMMON  = 2
	PRIVILEGE_TRUST   = 1
	PRIVILEGE_OWNER   = 0
)

type CreditItem struct {
	Point         int
	Credit        int
	Privilege     int
	ReceivedBytes uint64 // received bytes, or equivalent by valid payment
	SentBytes     uint64 // sent bytes
	Checkout      int
	LastAccess    time.Time
}

type CreditMap struct {
	lock *sync.RWMutex
	cmap map[string]CreditItem
}

func NewCreditManager() *CreditMap {
	return &CreditMap{
		lock: &sync.RWMutex{},
		cmap: make(map[string]CreditItem),
	}
}

func (c *CreditMap) SetUserCredit(key string, cdelta int) {

}

func (c *CreditMap) GetUserCredit(key string) CreditItem {
	var credit CreditItem

	return credit
}

func (c *CreditMap) CheckCredit(key, data, sign string) bool {
	return true
}
