package cache

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/CESSProject/cess-dcdn-components/cdn-node/types"
	"github.com/CESSProject/cess-dcdn-components/config"
	"github.com/CESSProject/cess-dcdn-components/logger"
	"github.com/CESSProject/cess-dcdn-components/protocol"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

var (
	BusyLine            = 1024
	MaxRequest          = 5
	MaxCachedSegmentNum = 16
	AvgCachedSegmentNum = 5
	MinCachedSegmentNum = 3
	DefaultNeighborNum  = 16
	MaxNeigborNum       = 64
)

func (c *Cacher) GetCacheInfo(userAcc string) types.CacheInfo {
	reqNum := c.GetRequestNumber()
	credit := c.cmap.GetUserCredit(userAcc)
	limit, _ := protocol.CheckPointLimit(
		credit, config.GetConfig().FreeDownloads, config.CACHE_BLOCK_SIZE, false,
	)
	info := types.CacheInfo{
		LoadRatio:    c.GetLoadRatio(),
		Account:      c.Account,
		Price:        c.Price,
		CreditLimit:  limit,
		CreditPoints: credit.Point,
		PeerId:       c.GetHost().ID().String(),
	}
	if reqNum > int64(BusyLine) && c.GetRequestNumber()-reqNum > 0 {
		info.Status = types.CACHE_BUSY
	} else {
		info.Status = types.CACHE_IDLE
	}
	return info
}

func (c *Cacher) ReadCacheService(req *pb.ReadfileRequest) (*pb.ReadfileResponse, error) {
	resp := &pb.ReadfileResponse{}
	extResp := types.QueryResponse{}

	data := req.GetExtendData()
	var extReq types.Request
	err := json.Unmarshal(data, &extReq)
	if err != nil {
		resp.Code = core.P2PResponseFailed
		logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
		return resp, errors.Wrap(err, "read cache service error")
	}
	defer func() {
		if err != nil {
			log.Println("read cache service: err", err)
			logger.GetLogger(types.LOG_CACHE).Error("read cache service: error ", err)
		}
	}()
	log.Println("read cache service: request", extReq)
	logger.GetLogger(types.LOG_CACHE).Info("read cache service: ", extReq)

	userAcc := common.BytesToAddress(extReq.AccountId).Hex() //hex.EncodeToString()
	switch extReq.Option {
	case types.OPTION_DAIL:
		extResp.Info = c.GetCacheInfo(userAcc)
		extResp.Status = types.STATUS_OK
		res, err := json.Marshal(extResp)
		if err != nil {
			resp.Code = core.P2PResponseRemoteFailed
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
			return resp, errors.Wrap(err, "read cache service error")
		}
		resp.Data = res
		resp.Length = uint32(len(res))
		resp.Code = core.P2PResponseFinish
	case types.OPTION_QUERY:
		extResp.Info = c.GetCacheInfo(userAcc)
		func() {
			if !c.cmap.CheckCredit(userAcc, extReq.Data, extReq.Sign) {
				extResp.Status = types.STATUS_FROZEN
				return
			}
			if req.Datahash == "" && req.Roothash == "" {
				extResp.Status = types.STATUS_OK
				return
			}
			if _, ok := c.taskQueue.Load(req.Datahash); ok {
				extResp.Status = types.STATUS_LOADING
				return
			}
			if fragments := c.QuerySegmentInfo(
				userAcc, req.Roothash, req.Datahash, MaxRequest); len(fragments) > 0 {
				extResp.Status = types.STATUS_HIT
				extResp.CachedFiles = fragments
				return
			}
			extResp.Status = types.STATUS_MISS
		}()

		res, err := json.Marshal(extResp)
		if err != nil {
			resp.Code = core.P2PResponseRemoteFailed
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
			return resp, errors.Wrap(err, "read cache service error")
		}
		resp.Data = res
		resp.Length = uint32(len(res))
		resp.Code = core.P2PResponseFinish
	case types.OPTION_DOWNLOAD:
		if !c.cmap.CheckCredit(userAcc, extReq.Data, extReq.Sign) {
			resp.Code = core.P2PResponseFailed
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
			return resp, errors.Wrap(errors.New("not enough credit points"), "read cache service error")
		}
		fname := filepath.Join(req.Roothash, req.Datahash, extReq.WantFile)
		fpath, err := c.GetCacheRecord(fname)
		if err != nil {
			resp.Code = core.P2PResponseFailed
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
			return resp, errors.Wrap(err, "read cache service error")
		}
		if _, err := os.Stat(fpath); err != nil {
			resp.Code = core.P2PResponseRemoteFailed
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
			return resp, errors.Wrap(err, "read cache service error")
		}
		f, err := os.Open(fpath)
		if err != nil {
			resp.Code = core.P2PResponseRemoteFailed
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
			return resp, errors.Wrap(err, "read cache service error")
		}
		defer f.Close()

		fstat, err := f.Stat()
		if err != nil {
			resp.Code = core.P2PResponseRemoteFailed
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
			return resp, errors.Wrap(err, "read cache service error")
		}

		if _, err = f.Seek(req.Offset, 0); err != nil {
			resp.Code = core.P2PResponseRemoteFailed
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
			return resp, errors.Wrap(err, "read cache service error")
		}

		var readBuf = make([]byte, core.FileProtocolBufSize)
		num, err := f.Read(readBuf)
		if err != nil {
			resp.Code = core.P2PResponseRemoteFailed
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
			return resp, errors.Wrap(err, "read cache service error")
		}

		if num+int(req.Offset) >= int(fstat.Size()) {
			resp.Code = core.P2PResponseFinish
			//record crdit bill
		}
		c.cmap.SetUserCredit(userAcc, -1)
		resp.Data = readBuf[:num]
		resp.Length = uint32(num)
		resp.Offset = req.Offset
	default:
		resp.Code = core.P2PResponseFailed
	}
	return resp, nil
}
