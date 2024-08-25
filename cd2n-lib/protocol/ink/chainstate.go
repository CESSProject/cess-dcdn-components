/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package ink

import (
	"log"

	"github.com/CESSProject/cess-go-sdk/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// GetPublicKey returns your own public key
func (c *chainClient) GetPublicKey() []byte {
	return c.keyring.PublicKey
}

// GetPublicKey returns your own public key
func (c *chainClient) GetIncomeAccount() string {
	return c.IncomeAcc
}

func (c *chainClient) GetCessAccount() (string, error) {
	return utils.EncodePublicKeyAsCessAccount(c.keyring.PublicKey)
}

func (c *chainClient) GetSyncStatus() (bool, error) {
	if !c.IsChainClientOk() {
		return false, ERR_RPC_CONNECTION
	}
	h, err := c.api.RPC.System.Health()
	if err != nil {
		return false, err
	}
	return h.IsSyncing, nil
}

func (c *chainClient) GetStorageFromChain(target any, prefix, method string, args ...[]byte) error {
	defer func() {
		recover()
	}()
	if !c.IsChainClientOk() {
		c.SetChainState(false)
		return ERR_RPC_CONNECTION
	}
	c.SetChainState(true)
	key, err := types.CreateStorageKey(c.metadata, prefix, method, args...)
	if err != nil {
		return errors.Wrap(err, "get storage from chain error")
	}
	ok, err := c.api.RPC.State.GetStorageLatest(key, target)
	if err != nil {
		return errors.Wrap(err, "get storage from chain error")
	} else if !ok {
		return errors.Wrap(errors.New(ERR_Empty), "get storage from chain error")
	}
	return nil
}

// Query file meta info
func (c *chainClient) GetFileMetaInfo(fid string) (FileMetaInfo, error) {
	var (
		data FileMetaInfo
		hash FileHash
	)
	if len(fid) != len(hash) {
		return data, errors.New(ERR_Failed)
	}
	for i := 0; i < len(hash); i++ {
		hash[i] = types.U8(fid[i])
	}

	b, err := types.Encode(hash)
	if err != nil {
		return data, errors.Wrap(err, "get file metadata error")
	}
	err = c.GetStorageFromChain(&data, _FILEBANK, _FILEMAP_FILEMETA, b)
	if err != nil {
		return data, errors.Wrap(err, "get file metadata error")
	}
	if len(data.BlockInfo) == 0 || data.Size == 0 {
		return data, errors.Wrap(errors.New("invalid file metadata"), "get file metadata error")
	}
	return data, nil
}

func (c *chainClient) GetAccountInfo() (types.AccountInfo, error) {
	var info types.AccountInfo
	err := c.GetStorageFromChain(
		&info,
		_SYSTEM,
		_SYSTEM_ACCOUNT,
		c.keyring.PublicKey,
	)
	if err != nil {
		return info, errors.Wrap(err, "get account info error")
	}
	return info, nil
}

func (c *chainClient) GetCachers() ([]CacherInfo, error) {
	var list []CacherInfo
	key := createPrefixedKey(_CACHER_CACHER, _CACHER)
	keys, err := c.api.RPC.State.GetKeysLatest(key)
	if err != nil {
		return list, errors.Wrap(err, "get cachers info error")
	}
	set, err := c.api.RPC.State.QueryStorageAtLatest(keys)
	if err != nil {
		return list, errors.Wrap(err, "get cachers info error")
	}
	for _, elem := range set {
		for _, change := range elem.Changes {
			var cacher CacherInfo
			if err := types.Decode(change.StorageData, &cacher); err != nil {
				log.Println(err)
				continue
			}
			list = append(list, cacher)
		}
	}
	return list, nil
}

func createPrefixedKey(method, prefix string) []byte {
	return append(xxhash.New128([]byte(prefix)).Sum(nil), xxhash.New128([]byte(method)).Sum(nil)...)
}
