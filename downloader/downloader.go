package downloader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-dcdn-components/light-cacher/ctype"
	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-go-sdk/config"
	"github.com/CESSProject/cess-go-sdk/core/crypte"
	"github.com/CESSProject/cess-go-sdk/core/erasure"
	"github.com/CESSProject/p2p-go/core"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
)

func DailCacheNode(peerNode *core.PeerNode, peerId peer.ID) (ctype.QueryResponse, error) {
	var resp ctype.QueryResponse
	buf := bytes.NewBuffer([]byte{})
	req := ctype.Request{
		Option: ctype.OPTION_DAIL,
	}
	err := SendRequestToCacher(peerNode, peerId, "", "", req, buf)
	if err != nil {
		return resp, errors.Wrap(err, "dail cache node error")
	}
	if buf.Len() == 0 {
		err = errors.New("empty response")
		return resp, errors.Wrap(err, "dail cache node error")
	}
	err = json.Unmarshal(buf.Bytes(), &resp)
	if err != nil {
		return resp, errors.Wrap(err, "dail cache node error")
	}
	return resp, nil
}

func QueryFileInfoFromCache(peerNode *core.PeerNode, acc []byte, peerId peer.ID, fileHash, segmentHash, data, sign string) (ctype.QueryResponse, error) {
	var resp ctype.QueryResponse
	buf := bytes.NewBuffer([]byte{})
	req := ctype.Request{
		Option:    ctype.OPTION_QUERY,
		AccountId: acc,
		Data:      []byte(data),
		Sign:      []byte(sign),
	}
	err := SendRequestToCacher(peerNode, peerId, fileHash, segmentHash, req, buf)
	if err != nil {
		return resp, errors.Wrap(err, "query file info error")
	}
	if buf.Len() == 0 {
		err = errors.New("empty response")
		return resp, errors.Wrap(err, "query file info error")
	}
	err = json.Unmarshal(buf.Bytes(), &resp)
	if err != nil {
		return resp, errors.Wrap(err, "query file info error")
	}
	return resp, nil
}

func DownloadFileFromCache(peerNode *core.PeerNode, keyPair signature.KeyringPair, peerId peer.ID, fpath, fileHash, segmentHash, fragmentHash string) error {
	buf := bytes.NewBuffer([]byte{})
	req := ctype.Request{
		Option:    ctype.OPTION_DOWNLOAD,
		WantFile:  fragmentHash,
		AccountId: keyPair.PublicKey,
		Data:      []byte(time.Now().String()),
	}
	sign, err := signature.Sign([]byte(req.Data), keyPair.URI)
	if err != nil {
		return errors.Wrap(err, "download file error")
	}
	req.Sign = sign
	err = SendRequestToCacher(peerNode, peerId, fileHash, segmentHash, req, buf)
	if err != nil {
		return errors.Wrap(err, "download file error")
	}
	fLen := buf.Len()
	if fLen == 0 {
		err = errors.New("empty response")
		return errors.Wrap(err, "download file error")
	}
	f, err := os.Create(fpath)
	if err != nil {
		return errors.Wrap(err, "download file error")
	}
	f.Close()
	n, err := io.Copy(f, buf)
	if err != nil {
		return errors.Wrap(err, "download file error")
	}
	if n != int64(fLen) {
		err = fmt.Errorf("failed to write data, expected %d bytes, actual %d bytes", fLen, n)
		return errors.Wrap(err, "download file error")
	}
	return nil
}

func DownloadFragmentFromStorage(fpath, fileHash, segmentHash, fragmentHash string, chainCli *chain.ChainClient, peerNode *core.PeerNode) error {
	fmeta, err := chainCli.QueryFile(fileHash, -1)
	if err != nil {
		return errors.Wrap(err, "download fragment from storage error")
	}
	for _, segment := range fmeta.SegmentList {
		if string(segment.Hash[:]) != segmentHash {
			continue
		}
		for _, fragment := range segment.FragmentList {
			if string(fragment.Hash[:]) != fragmentHash {
				continue
			}
			miner, err := chainCli.QueryMinerItems(fragment.Miner[:], -1)
			if err != nil {
				return errors.Wrap(err, "download fragment from storage error")
			}
			err = peerNode.ReadFileAction(peer.ID(miner.PeerId[:]), fileHash, segmentHash, fpath, config.FragmentSize)
			if err != nil {
				return errors.Wrap(err, "download fragment from storage error")
			}
			break
		}
	}
	return nil
}

func DownloadSegmentFromStorage(fdir, fileHash, segmentHash string, chainCli *chain.ChainClient, peerNode *core.PeerNode) (string, error) {

	segmentPath := filepath.Join(fdir, segmentHash)
	fmeta, err := chainCli.QueryFile(fileHash, -1)
	if err != nil {
		return segmentPath, errors.Wrap(err, "download fragment from storage error")
	}
	count := 0
	paths := make([]string, 0)
	for _, segment := range fmeta.SegmentList {
		if string(segment.Hash[:]) != segmentHash {
			continue
		}
		for _, fragment := range segment.FragmentList {
			if count >= config.DataShards {
				break
			}
			miner, err := chainCli.QueryMinerItems(fragment.Miner[:], -1)
			if err != nil {
				continue
			}
			fragmentPath := filepath.Join(fdir, string(fragment.Hash[:]))
			if err = peerNode.ReadFileAction(
				peer.ID(miner.PeerId[:]), fileHash, segmentHash,
				fragmentPath, config.FragmentSize,
			); err != nil {
				continue
			}
			count++
			paths = append(paths, fragmentPath)
		}
	}
	if count < config.DataShards {
		err := errors.New("not enough fragments were downloaded")
		return segmentPath, errors.Wrap(err, "download segment from storage error")
	}
	err = erasure.RSRestore(segmentPath, paths)
	if err != nil {
		return segmentPath, errors.Wrap(err, "download segment from storage error")
	}
	for _, p := range paths {
		os.Remove(p)
	}
	return segmentPath, nil
}

func DownloadFileFromStorage(fdir, fileHash, cipher string, chainCli *chain.ChainClient, peerNode *core.PeerNode) (string, error) {

	if _, err := os.Stat(fdir); err != nil {
		return "", errors.Wrap(err, "download file from storage error")
	}
	userfile := filepath.Join(fdir, fileHash)
	f, err := os.Create(userfile)
	if err != nil {
		return "", errors.Wrap(err, "download file from storage error")
	}
	defer f.Close()

	fmeta, err := chainCli.QueryFile(fileHash, -1)
	if err != nil {
		return "", errors.Wrap(err, "download file from storage error")
	}

	defer func(basedir string) {
		for _, segment := range fmeta.SegmentList {
			os.Remove(filepath.Join(basedir, string(segment.Hash[:])))
			for _, fragment := range segment.FragmentList {
				os.Remove(filepath.Join(basedir, string(fragment.Hash[:])))
			}
		}
	}(fdir)

	segmentspath := make([]string, 0)
	for _, segment := range fmeta.SegmentList {
		fragmentpaths := make([]string, 0)
		for _, fragment := range segment.FragmentList {
			miner, err := chainCli.QueryMinerItems(fragment.Miner[:], -1)
			if err != nil {
				return "", errors.Wrap(err, "download file from storage error")
			}

			fragmentpath := filepath.Join(fdir, string(fragment.Hash[:]))
			err = peerNode.ReadFileAction(peer.ID(miner.PeerId[:]), fileHash, string(fragment.Hash[:]), fragmentpath, config.FragmentSize)
			if err != nil {
				continue
			}
			fragmentpaths = append(fragmentpaths, fragmentpath)
			segmentpath := filepath.Join(fdir, string(segment.Hash[:]))
			if len(fragmentpaths) >= config.DataShards {
				err = erasure.RSRestore(segmentpath, fragmentpaths)
				if err != nil {
					return "", err
				}
				segmentspath = append(segmentspath, segmentpath)
				break
			}
		}
	}

	if len(segmentspath) != len(fmeta.SegmentList) {
		err = errors.New("the number of downloaded segments is inconsistent")
		return "", errors.Wrap(err, "download file from storage error")
	}

	err = RecoveryFileViaSegments(segmentspath, fmeta, cipher, f)
	if err != nil {
		return "", errors.Wrap(err, "download file from storage error")
	}
	return userfile, nil
}

func SendRequestToCacher(handle *core.PeerNode, peerId peer.ID, roothash, datahash string, req ctype.Request, resp io.Writer) error {
	extData, err := json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "send request to cacher error")
	}
	err = handle.ReadFileActionWithExtension(peerId, "", "", resp, extData)
	if err != nil {
		return errors.Wrap(err, "send request to cacher error")
	}
	return nil
}

func RecoveryFileViaSegments(segmentspath []string, fmeta chain.FileMetadata, cipher string, writer io.Writer) error {
	var writecount = 0
	for i := 0; i < len(fmeta.SegmentList); i++ {
		for j := 0; j < len(segmentspath); j++ {
			if string(fmeta.SegmentList[i].Hash[:]) == filepath.Base(segmentspath[j]) {
				buf, err := os.ReadFile(segmentspath[j])
				if err != nil {
					return errors.Wrap(err, "file recovery via segments failed")
				}
				if cipher != "" {
					buf, err = crypte.AesCbcDecrypt(buf, []byte(cipher))
					if err != nil {
						return errors.Wrap(err, "file recovery via segments failed")
					}
				}
				if (writecount + 1) >= len(fmeta.SegmentList) {
					if cipher != "" {
						writer.Write(buf[:(fmeta.FileSize.Uint64() - uint64(writecount*(config.SegmentSize-16)))])
					} else {
						writer.Write(buf[:(fmeta.FileSize.Uint64() - uint64(writecount*config.SegmentSize))])
					}
				} else {
					writer.Write(buf)
				}
				writecount++
				break
			}
		}
	}
	if writecount != len(fmeta.SegmentList) {
		err := errors.New("the number of processed segments is inconsistent")
		return errors.Wrap(err, "file recovery via segments failed")
	}
	return nil
}
