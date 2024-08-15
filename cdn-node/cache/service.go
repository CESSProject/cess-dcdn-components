package cache

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/CESSProject/cess-dcdn-components/cdn-lib/types"
	"github.com/CESSProject/cess-dcdn-components/config"
	"github.com/CESSProject/cess-dcdn-components/logger"
	"github.com/CESSProject/cess-dcdn-components/protocol"
	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/network"
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

func WriteJsonResponse(s network.Stream, status string, data any) error {
	resp := types.Response{
		Status: status,
		Data:   data,
	}
	res, err := json.Marshal(resp)
	if err != nil {
		s.Reset()
		return err
	}
	if _, err = s.Write(res); err != nil {
		s.Reset()
		return err
	}
	return nil
}

func (c *Cacher) CacheService(s network.Stream) {
	defer s.Close()
	var (
		extReq  types.CacheRequest
		extResp types.CacheResponse
	)
	data, err := io.ReadAll(s)
	if err != nil {
		logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
		WriteJsonResponse(s, types.STATUS_ERROR, err.Error())
		return
	}
	err = json.Unmarshal(data, &extReq)
	if err != nil {
		logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
		WriteJsonResponse(s, types.STATUS_ERROR, err.Error())
		return
	}
	userAcc := common.BytesToAddress(extReq.AccountId).Hex() //hex.EncodeToString()

	paths := strings.Split(extReq.WantFile, "/")

	logger.GetLogger(types.LOG_CACHE).Info("read cache service: ", extReq.Option, " , ", userAcc)

	switch extReq.Option {
	case types.OPTION_DAIL:
		extResp.Info = c.GetCacheInfo(userAcc)
		extResp.Status = types.STATUS_OK
		WriteJsonResponse(s, types.STATUS_OK, extResp)
		return
	case types.OPTION_QUERY:
		extResp.Info = c.GetCacheInfo(userAcc)
		func() {
			if !c.cmap.CheckCredit(userAcc, extReq.Data, extReq.Sign) {
				extResp.Status = types.STATUS_FROZEN
				return
			}
			if len(paths) < 2 {
				extResp.Status = types.STATUS_OK
				return
			}
			if _, ok := c.taskQueue.Load(paths[1]); ok {
				extResp.Status = types.STATUS_LOADING
				return
			}
			if fragments := c.QuerySegmentInfo(
				userAcc, paths[0], paths[1], MaxRequest); len(fragments) > 0 {
				extResp.Status = types.STATUS_HIT
				extResp.CachedFiles = fragments
				return
			}
			extResp.Status = types.STATUS_MISS
		}()
		WriteJsonResponse(s, types.STATUS_OK, extResp)
		return
	case types.OPTION_DOWNLOAD:
		if len(paths) < 3 {
			WriteJsonResponse(s, types.STATUS_ERROR, "illegal file Id and segment Id")
			return
		}
		if !c.cmap.CheckCredit(userAcc, extReq.Data, extReq.Sign) {
			WriteJsonResponse(s, types.STATUS_ERROR, "failed credit check")
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", "failed credit check")
			return
		}
		fname := filepath.Join(paths[0], paths[1], paths[2])
		fpath, err := c.GetCacheRecord(fname)
		if err != nil {
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
			WriteJsonResponse(s, types.STATUS_ERROR, err.Error())
			return
		}
		if _, err := os.Stat(fpath); err != nil {
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
			WriteJsonResponse(s, types.STATUS_ERROR, err.Error())
			return
		}
		f, err := os.Open(fpath)
		if err != nil {
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
			WriteJsonResponse(s, types.STATUS_ERROR, err.Error())
			return
		}
		defer f.Close()
		bytes, err := io.ReadAll(f)
		if err != nil {
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
			WriteJsonResponse(s, types.STATUS_ERROR, err.Error())
			return
		}
		_, err = s.Write(bytes)
		if err != nil {
			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
			s.Reset()
		}

	default:
		s.Reset()
	}
}

// func (c *Cacher) ReadCacheService(req *pb.ReadfileRequest) (*pb.ReadfileResponse, error) {
// 	resp := &pb.ReadfileResponse{}
// 	extResp := types.CacheResponse{}

// 	data := req.GetExtendData()
// 	var extReq types.CacheRequest
// 	err := json.Unmarshal(data, &extReq)
// 	if err != nil {
// 		resp.Code = core.P2PResponseFailed
// 		logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
// 		return resp, errors.Wrap(err, "read cache service error")
// 	}
// 	defer func() {
// 		if err != nil {
// 			log.Println("read cache service: err", err)
// 			logger.GetLogger(types.LOG_CACHE).Error("read cache service: error ", err)
// 		}
// 	}()

// 	userAcc := common.BytesToAddress(extReq.AccountId).Hex() //hex.EncodeToString()

// 	logger.GetLogger(types.LOG_CACHE).Info("read cache service: ", extReq.Option, " , ", userAcc)

// 	switch extReq.Option {
// 	case types.OPTION_DAIL:

// 		extResp.Info = c.GetCacheInfo(userAcc)
// 		extResp.Status = types.STATUS_OK
// 		res, err := json.Marshal(extResp)
// 		if err != nil {
// 			resp.Code = core.P2PResponseRemoteFailed
// 			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
// 			return resp, errors.Wrap(err, "read cache service error")
// 		}
// 		resp.Data = res
// 		resp.Length = uint32(len(res))
// 		resp.Code = core.P2PResponseFinish
// 	case types.OPTION_QUERY:
// 		extResp.Info = c.GetCacheInfo(userAcc)
// 		func() {
// 			if !c.cmap.CheckCredit(userAcc, extReq.Data, extReq.Sign) {
// 				extResp.Status = types.STATUS_FROZEN
// 				return
// 			}
// 			if req.Datahash == "" && req.Roothash == "" {
// 				extResp.Status = types.STATUS_OK
// 				return
// 			}
// 			if _, ok := c.taskQueue.Load(req.Datahash); ok {
// 				extResp.Status = types.STATUS_LOADING
// 				return
// 			}
// 			if fragments := c.QuerySegmentInfo(
// 				userAcc, req.Roothash, req.Datahash, MaxRequest); len(fragments) > 0 {
// 				extResp.Status = types.STATUS_HIT
// 				extResp.CachedFiles = fragments
// 				return
// 			}
// 			extResp.Status = types.STATUS_MISS
// 		}()

// 		res, err := json.Marshal(extResp)
// 		if err != nil {
// 			resp.Code = core.P2PResponseRemoteFailed
// 			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
// 			return resp, errors.Wrap(err, "read cache service error")
// 		}
// 		resp.Data = res
// 		resp.Length = uint32(len(res))
// 		resp.Code = core.P2PResponseFinish
// 	case types.OPTION_DOWNLOAD:
// 		if !c.cmap.CheckCredit(userAcc, extReq.Data, extReq.Sign) {
// 			resp.Code = core.P2PResponseFailed
// 			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
// 			return resp, errors.Wrap(errors.New("not enough credit points"), "read cache service error")
// 		}
// 		fname := filepath.Join(req.Roothash, req.Datahash, extReq.WantFile)
// 		fpath, err := c.GetCacheRecord(fname)
// 		if err != nil {
// 			resp.Code = core.P2PResponseFailed
// 			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
// 			return resp, errors.Wrap(err, "read cache service error")
// 		}
// 		if _, err := os.Stat(fpath); err != nil {
// 			resp.Code = core.P2PResponseRemoteFailed
// 			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
// 			return resp, errors.Wrap(err, "read cache service error")
// 		}
// 		f, err := os.Open(fpath)
// 		if err != nil {
// 			resp.Code = core.P2PResponseRemoteFailed
// 			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
// 			return resp, errors.Wrap(err, "read cache service error")
// 		}
// 		defer f.Close()

// 		fstat, err := f.Stat()
// 		if err != nil {
// 			resp.Code = core.P2PResponseRemoteFailed
// 			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
// 			return resp, errors.Wrap(err, "read cache service error")
// 		}

// 		if _, err = f.Seek(req.Offset, 0); err != nil {
// 			resp.Code = core.P2PResponseRemoteFailed
// 			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
// 			return resp, errors.Wrap(err, "read cache service error")
// 		}

// 		var readBuf = make([]byte, core.FileProtocolBufSize)
// 		num, err := f.Read(readBuf)
// 		if err != nil {
// 			resp.Code = core.P2PResponseRemoteFailed
// 			logger.GetLogger(types.LOG_CACHE).Error("read cache service error", err)
// 			return resp, errors.Wrap(err, "read cache service error")
// 		}

// 		if num+int(req.Offset) >= int(fstat.Size()) {
// 			resp.Code = core.P2PResponseFinish
// 			//record crdit bill
// 		}
// 		c.cmap.SetUserCredit(userAcc, -1)
// 		resp.Data = readBuf[:num]
// 		resp.Length = uint32(num)
// 		resp.Offset = req.Offset
// 	default:
// 		resp.Code = core.P2PResponseFailed
// 	}
// 	return resp, nil
// }
