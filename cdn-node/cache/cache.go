package cache

import (
	"context"
	"encoding/binary"
	"io/fs"
	"math/rand"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cdnlib "github.com/CESSProject/cess-dcdn-components/cdn-lib"
	"github.com/CESSProject/cess-dcdn-components/cdn-lib/types"
	"github.com/CESSProject/cess-dcdn-components/logger"
	"github.com/CESSProject/cess-dcdn-components/p2p"
	"github.com/CESSProject/cess-dcdn-components/protocol"
	"github.com/CESSProject/cess-dcdn-components/protocol/contract"
	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-go-sdk/config"
	"github.com/CESSProject/cess-go-tools/cacher"
	"github.com/CESSProject/cess-go-tools/scheduler"
	"github.com/CESSProject/p2p-go/core"
	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type Cacher struct {
	*chain.ChainClient
	cacher.FileCache
	scheduler.Selector
	*core.PeerNode
	reqNum    *atomic.Int64
	dlQueue   *sync.Map //download queue
	taskQueue *sync.Map // download task queue
	record    *leveldb.DB
	cmap      *protocol.CreditMap
	Account   []byte
	Price     string
}

func NewCacher(sdk *chain.ChainClient, cache cacher.FileCache, p2pNode *core.PeerNode, selector scheduler.Selector, cli *contract.Client) *Cacher {
	if sdk == nil || cache == nil || p2pNode == nil {
		return nil
	}
	c := &Cacher{
		Selector:    selector,
		ChainClient: sdk,
		FileCache:   cache,
		PeerNode:    p2pNode,
		taskQueue:   &sync.Map{},
		reqNum:      &atomic.Int64{},
		cmap:        protocol.NewCreditManager(cli),
	}
	db, err := leveldb.OpenFile("./file_record", nil)
	if err != nil {
		return nil
	}
	c.record = db

	c.AddCallbackOfAddItem(func(item cacher.CacheItem) {
		k := item.Key().(string)
		k, _ = filepath.Split(k)
		buf := make([]byte, 4)
		v, err := c.record.Get([]byte(k+types.RECORD_FRAGMENTS), nil)
		if err != nil {
			if err == leveldb.ErrNotFound {
				binary.BigEndian.PutUint32(buf, 1)
				c.record.Put([]byte(k+types.RECORD_FRAGMENTS), buf, nil)
			}
		} else {
			if len(v) < 4 {
				binary.BigEndian.PutUint32(buf, 1)
			} else {
				binary.BigEndian.PutUint32(buf, 1+binary.BigEndian.Uint32(v))
			}
			c.record.Put([]byte(k+types.RECORD_FRAGMENTS), buf, nil)
		}
	})

	c.AddCallbackOfDeleteItem(func(item cacher.CacheItem) {
		k := item.Key().(string)
		k, _ = filepath.Split(k)
		v, err := c.record.Get([]byte(k+types.RECORD_FRAGMENTS), nil)

		if err == nil && len(v) >= 4 {
			value := binary.BigEndian.Uint32(v[:4])
			if value > 0 {
				value -= 1
			}
			binary.BigEndian.PutUint32(v[:4], value)
			c.record.Put([]byte(k+types.RECORD_FRAGMENTS), v[:4], nil)
			if value < config.DataShards {
				buf := make([]byte, 4)
				binary.BigEndian.PutUint32(buf, 0)
				c.record.Put([]byte(k+types.RECORD_REQUESTS), buf, nil)
			}
		}
	})

	return c
}

func (c *Cacher) SetConfig(price string, Acc []byte) {
	c.Price = price
	c.Account = Acc
}

func (c *Cacher) RunDiscovery(ctx context.Context, bootNode string) error {

	var err error
	ch := make(chan peer.AddrInfo, 64)
	ticker := time.NewTicker(time.Hour * 3)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case peer := <-ch:
				if peer.ID.Size() == 0 {
					break
				}
				c.Selector.FlushPeerNodes(5*time.Second, peer)
				//logger.GetLogger(types.LOG_CACHE).Info("flush cache node peer", peer.ID)
			case <-ticker.C:
			}
		}
	}()
	go func() {
		err = p2p.StartDiscoveryFromMDNS(ctx, c.GetHost(), ch)
	}()
	err = p2p.StartDiscoveryFromDHT(
		context.Background(),
		c.GetHost(),
		c.GetDHTable(),
		p2p.DISCOVERY_RENDEZVOUS,
		time.Second*3, ch,
	)
	if err != nil {
		return errors.Wrap(err, "run discovery service error")
	}
	return nil
}

func (c *Cacher) GetFileRecord(key, rType string) (int, bool) {
	c.GetRendezvousVersion()
	if rType != types.RECORD_FRAGMENTS && rType != types.RECORD_REQUESTS {
		return 0, false
	}
	v, err := c.record.Get([]byte(key+rType), nil)
	if err != nil {
		return 0, false
	}
	if len(v) < 4 {
		return 0, false
	}
	return int(binary.BigEndian.Uint32(v[:4])), true
}

func (c *Cacher) PutFileRecord(key, rType string, value int) error {
	if rType != types.RECORD_FRAGMENTS && rType != types.RECORD_REQUESTS {
		return errors.New("bad record type")
	}
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(value))
	return c.record.Put([]byte(key+rType), buf, nil)
}

func (c *Cacher) RestoreCacheFiles(cacheDir string) error {
	return filepath.Walk(cacheDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Size() != config.FragmentSize {
			return nil
		}
		paths := strings.Split(path, "/")
		l := len(paths)
		if l < 4 {
			return nil
		}
		c.AddCacheRecord(filepath.Join(paths[l-1], paths[l-2], paths[l-3]), path)
		return nil
	})
}

func (c *Cacher) RunDownloadServer(ctx context.Context, threadNum int) {
	if c.dlQueue != nil {
		return
	}
	if threadNum <= 0 {
		threadNum = runtime.NumCPU()/4 + 1
	}
	c.dlQueue = &sync.Map{}
	defer func() { c.dlQueue = nil }()
	wg := &sync.WaitGroup{}
	for i := 0; i < threadNum; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep((5*time.Duration(idx) + 15) * time.Second)
				}
				c.dlQueue.Range(func(key, value any) bool {
					if !c.dlQueue.CompareAndSwap(key, false, true) {
						return true
					}
					//download file
					k := key.(string)
					hashs := strings.Split(k, "/")
					if len(hashs) != 2 {
						return true
					}
					count := 0
					v, err := c.record.Get([]byte(k+types.RECORD_FRAGMENTS), nil)
					if err == nil && len(v) >= 4 {
						count = int(binary.BigEndian.Uint32(v[:4]))
					}
					if count > config.DataShards {
						c.dlQueue.CompareAndDelete(key, true)
						return true
					}
					err = c.DownloadFiles(hashs[0], hashs[1])
					if err != nil {
						logger.GetLogger(types.LOG_CACHE).Error("download files error", err)
						c.dlQueue.CompareAndSwap(key, true, false)
					}
					c.dlQueue.CompareAndDelete(key, true)
					return true
				})
			}
		}()
	}
	wg.Wait()
}

func (c *Cacher) RegisterP2pTsFileServiceHandle(handle core.ReadFileServerHandle) {
	c.SetReadFileServiceHandle(handle)
}

func (c *Cacher) SetRequestNumber(delta int64) {
	c.reqNum.Add(delta)
}

func (c *Cacher) GetRequestNumber() int64 {
	return c.reqNum.Load()
}

func (c *Cacher) DownloadFiles(fhash, shash string) error {
	logger.GetLogger(types.LOG_CACHE).Info("download files ", fhash, " ", shash)
	fmeta, err := c.QueryFile(fhash, -1)
	if err != nil {
		return errors.Wrap(err, "download fragments from storage miners error")
	}
	cacheNum := MinCachedSegmentNum + len(fmeta.SegmentList)/DefaultNeighborNum
	if cacheNum > MaxCachedSegmentNum {
		cacheNum = MaxCachedSegmentNum
	}

	ShuffleSegments(fmeta.SegmentList)
	num, done := 0, false
	for _, segment := range fmeta.SegmentList {
		if num >= cacheNum {
			break
		}
		//query file from cache node
		if string(segment.Hash[:]) == shash {
			done = true
		}
		if !done && num >= cacheNum-1 {
			continue
		}
		cachedPeers, err := c.QuerySegmentFromNeighborCacher(fhash, string(segment.Hash[:]))
		logger.GetLogger(types.LOG_CACHE).Info("query segment from neighbor cache node ", cachedPeers)
		if err == nil {
			if len(cachedPeers) >= AvgCachedSegmentNum {
				continue
			}
			//shared file from cache node
			logger.GetLogger(types.LOG_CACHE).Info("shared segment from neighbor ")
			if c.SharedSegmentFromNeighborCacher(fhash, segment, cachedPeers) {
				logger.GetLogger(types.LOG_CACHE).Info("fetched segment from neighbor")
				num++
				continue
			}
		}
		count := 0
		logger.GetLogger(types.LOG_CACHE).Info("download segment from storage node ")
		for _, fragment := range segment.FragmentList {
			if count >= config.DataShards {
				return nil
			}
			hash := string(fragment.Hash[:])
			fname := filepath.Join(fhash, shash, hash)
			cpath, err := c.GetCacheRecord(fname)
			if err == nil && cpath != "" { //fragment already exist
				count++
				logger.GetLogger(types.LOG_CACHE).Info("get fragment in local cache")
				continue
			}

			miner, err := c.QueryMinerItems(fragment.Miner[:], -1)
			if err != nil {
				logger.GetLogger(types.LOG_CACHE).Error("query miner items error ", err)
				continue
			}
			// bk := base58.Encode([]byte(string(miner.PeerId[:])))
			fpath := filepath.Join(types.TempDir, hash)
			//
			logger.GetLogger(types.LOG_CACHE).Infof("read file %s from storage node %v", hash, peer.ID(string(miner.PeerId[:])).String())
			err = c.ReadFileAction(peer.ID(string(miner.PeerId[:])), fhash, hash, fpath, config.FragmentSize)
			if err != nil {
				logger.GetLogger(types.LOG_CACHE).Errorf("read file %s from storage node %v error %v", hash, peer.ID(string(miner.PeerId[:])).String(), err)
				continue
			}
			logger.GetLogger(types.LOG_CACHE).Info("cache file: ", fname, " path: ", fpath)
			err = c.MoveFileToCache(fname, fpath)
			if err != nil {
				logger.GetLogger(types.LOG_CACHE).Errorf("move file %s to cache error %v", fname, err)
				continue
			}
			count++
		}
		num++
	}
	return nil
}

func (c *Cacher) QuerySegmentInfo(acc, fhash, shash string, maxReq int) []string {
	logger.GetLogger(types.LOG_CACHE).Infof("query segment info in cache, user: %s, file: %s, segment: %s", acc, fhash, shash)
	uc := c.cmap.GetUserCredit(acc)
	if uc.Perm > protocol.PERM_COMMON {
		return nil
	}

	k := filepath.Join(fhash, shash)
	count, req := 0, 0
	count, _ = c.GetFileRecord(k, types.RECORD_FRAGMENTS)
	req, _ = c.GetFileRecord(k, types.RECORD_REQUESTS)
	req += 1
	c.PutFileRecord(k, types.RECORD_REQUESTS, req)
	//query all file
	itor := c.record.NewIterator(util.BytesPrefix([]byte(fhash)), nil)
	cachedSegs := 0
	for itor.Next() {
		k := itor.Key()
		if strings.Contains(string(k), types.RECORD_REQUESTS) {
			continue
		}
		v := itor.Value()
		if len(v) < 4 {
			continue
		}
		if binary.BigEndian.Uint32(v[:4]) > 0 {
			cachedSegs++
		}
	}
	itor.Release()

	r := c.FileCache.GetLoadRatio()
	if (uc.Perm < protocol.PERM_COMMON || cachedSegs < MaxCachedSegmentNum) &&
		(req >= maxReq || r < 0.8 || uc.Perm == 0) &&
		count < config.DataShards && c.dlQueue != nil {
		_, ok := c.dlQueue.LoadOrStore(k, false)
		logger.GetLogger(types.LOG_CACHE).Info("try to queue the data for download ", !ok)
	}

	if count <= 0 {
		return nil
	}
	var res []string
	c.TraverseCache(func(key interface{}, item cacher.CacheItem) {
		itemKey := key.(string)
		if strings.Contains(itemKey, k) {
			_, file := filepath.Split(itemKey)
			res = append(res, file)
		}
	})
	logger.GetLogger(types.LOG_CACHE).Info("query cached files ", res)
	return res
}

func (c *Cacher) QuerySegmentFromNeighborCacher(fileHash, segmentHash string) (map[peer.ID]types.CacheResponse, error) {
	cachedFilePeers := make(map[peer.ID]types.CacheResponse)
	peerNum := c.Selector.GetPeersNumber()
	if peerNum <= 0 {
		return nil, errors.New("no neighbor cache node")
	}
	if peerNum > MaxNeigborNum {
		peerNum = MaxNeigborNum
	}
	threadNum := DefaultNeighborNum
	if peerNum < threadNum {
		threadNum = peerNum
	}
	itor, err := c.Selector.NewPeersIterator(peerNum)
	if err != nil {
		return nil, err
	}
	wg := sync.WaitGroup{}
	lock := sync.Mutex{}
	wg.Add(threadNum)
	for i := 0; i < threadNum; i++ {
		go func() {
			defer wg.Done()
			for {
				lock.Lock()
				peer, ok := itor.GetPeer()
				lock.Unlock()
				if !ok {
					return
				}
				resp, err := cdnlib.QueryFileInfoFromCache(
					c.PeerNode, peer.ID,
					&cdnlib.Options{
						Account:  c.Account,
						WantFile: path.Join(fileHash, segmentHash),
					},
				)
				if err != nil {
					c.Selector.Feedback(peer.ID.String(), false)
					continue
				}
				c.Selector.Feedback(peer.ID.String(), true)
				if resp.Status != types.STATUS_HIT && resp.Status != types.STATUS_LOADING {
					continue
				}
				lock.Lock()
				if len(cachedFilePeers) >= AvgCachedSegmentNum {
					lock.Unlock()
					return
				}
				cachedFilePeers[peer.ID] = resp
				lock.Unlock()
			}
		}()
	}
	wg.Wait()
	return cachedFilePeers, nil
}

func (c *Cacher) SharedSegmentFromNeighborCacher(fhash string, segment chain.SegmentInfo, peers map[peer.ID]types.CacheResponse) bool {
	cachedFragments := map[string]struct{}{}
	shash := string(segment.Hash[:])
	count := 0
	for id, res := range peers {
		if res.Status == types.STATUS_LOADING {
			continue
		}
		if count >= config.DataShards {
			break
		}
		for i := 0; i < len(res.CachedFiles); i++ {
			if _, ok := cachedFragments[res.CachedFiles[i]]; ok {
				continue
			}
			fpath := filepath.Join(types.TempDir, res.CachedFiles[i])
			err := cdnlib.DownloadFileFromCache(
				c.PeerNode, id, fpath,
				&cdnlib.Options{
					WantFile: path.Join(fhash, string(segment.Hash[:]), res.CachedFiles[i]),
					Account:  c.Account,
				},
			)
			if err != nil {
				continue
			}
			acc := common.BytesToAddress(res.Info.Account).Hex()
			c.cmap.SetUserCredit(acc, +1)
			err = c.FileCache.MoveFileToCache(filepath.Join(fhash, shash, res.CachedFiles[i]), fpath)
			if err != nil {
				continue
			}
			count++
			cachedFragments[res.CachedFiles[i]] = struct{}{}
		}

	}
	return count >= config.DataShards
}

func ShuffleSegments(segments []chain.SegmentInfo) {
	for i := len(segments) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		segments[i], segments[j] = segments[j], segments[i]
	}
}
