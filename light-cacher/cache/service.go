package cache

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/CESSProject/cess-dcdn-components/config"
	"github.com/CESSProject/cess-dcdn-components/light-cacher/ctype"
	"github.com/CESSProject/cess-dcdn-components/protocol"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/pb"
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

func (c *Cacher) ReadCacheService(req *pb.ReadfileRequest) (*pb.ReadfileResponse, error) {
	reqNum := c.GetRequestNumber()
	resp := &pb.ReadfileResponse{}
	extResp := ctype.QueryResponse{}

	data := req.GetExtendData()
	var extReq ctype.Request
	err := json.Unmarshal(data, &extReq)
	if err != nil {
		resp.Code = core.P2PResponseFailed
		return resp, errors.Wrap(err, "read cache service error")
	}

	userAcc := hex.EncodeToString(extReq.AccountId)
	switch extReq.Option {
	case ctype.OPTION_DAIL:
		credit := c.cmap.GetUserCredit(userAcc)
		limit, _ := protocol.CheckPoint(credit, config.GetConfig().FreeDownloads, config.CACHE_BLOCK_SIZE)
		info := ctype.CacheInfo{
			LoadRatio:    c.GetLoadRatio(),
			Account:      c.Account,
			Price:        c.Price,
			CreditLimit:  limit,
			CreditPoints: credit.Point,
		}
		if reqNum > int64(BusyLine) && c.GetRequestNumber()-reqNum > 0 {
			info.Status = ctype.CACHE_BUSY
		} else {
			info.Status = ctype.CACHE_IDLE
		}
		extResp.Info = &info
		extResp.Status = ctype.STATUS_OK
		res, err := json.Marshal(extResp)
		if err != nil {
			resp.Code = core.P2PResponseRemoteFailed
			return resp, errors.Wrap(err, "read cache service error")
		}
		resp.Data = res
		resp.Length = uint32(len(res))
		resp.Code = core.P2PResponseFinish
	case ctype.OPTION_QUERY:
		if !c.cmap.CheckCredit(userAcc, extReq.Data, extReq.Sign) {
			extResp.Status = ctype.STATUS_FROZEN
		} else if _, ok := c.taskQueue.Load(req.Datahash); ok {
			extResp.Status = ctype.STATUS_LOADING
		} else if fragments := c.QuerySegmentInfo(
			userAcc, req.Roothash, req.Datahash, MaxRequest); len(fragments) <= 0 {
			extResp.Status = ctype.STATUS_MISS
		} else {
			extResp.Status = ctype.STATUS_HIT
			extResp.CachedFiles = fragments
		}
		res, err := json.Marshal(extResp)
		if err != nil {
			resp.Code = core.P2PResponseRemoteFailed
			return resp, errors.Wrap(err, "read cache service error")
		}
		resp.Data = res
		resp.Length = uint32(len(res))
		resp.Code = core.P2PResponseFinish
	case ctype.OPTION_DOWNLOAD:
		if !c.cmap.CheckCredit(userAcc, extReq.Data, extReq.Sign) {
			resp.Code = core.P2PResponseFailed
			return resp, errors.Wrap(errors.New("not enough credit points"), "read cache service error")
		}
		fname := filepath.Join(req.Roothash, req.Datahash, extReq.WantFile)
		fpath, err := c.GetCacheRecord(fname)
		if err != nil {
			resp.Code = core.P2PResponseFailed
			return resp, errors.Wrap(err, "read cache service error")
		}
		if _, err := os.Stat(fpath); err != nil {
			resp.Code = core.P2PResponseRemoteFailed
			return resp, errors.Wrap(err, "read cache service error")
		}
		f, err := os.Open(fpath)
		if err != nil {
			resp.Code = core.P2PResponseRemoteFailed
			return resp, errors.Wrap(err, "read cache service error")
		}
		defer f.Close()

		fstat, err := f.Stat()
		if err != nil {
			resp.Code = core.P2PResponseRemoteFailed
			return resp, errors.Wrap(err, "read cache service error")
		}

		if _, err = f.Seek(req.Offset, 0); err != nil {
			resp.Code = core.P2PResponseRemoteFailed
			return resp, errors.Wrap(err, "read cache service error")
		}

		var readBuf = make([]byte, core.FileProtocolBufSize)
		num, err := f.Read(readBuf)
		if err != nil {
			resp.Code = core.P2PResponseRemoteFailed
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
