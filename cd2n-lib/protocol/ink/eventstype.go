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

import "github.com/centrifuge/go-substrate-rpc-client/v4/types"

//**************************************************************

type CacherRegister struct {
	Phase  types.Phase
	Acc    types.AccountID
	Info   CacherInfo
	Topics []types.Hash
}

type CacherLogout struct {
	Phase  types.Phase
	Acc    types.AccountID
	Topics []types.Hash
}

type Pay struct {
	Phase  types.Phase
	Acc    types.AccountID
	Bills  []Bill
	Topics []types.Hash
}

type CacheEventRecords struct {
	types.EventRecords
	Cacher_Register []CacherRegister
	Cacher_Update   []CacherRegister
	Cacher_Logout   []CacherLogout
	Cacher_Pay      []Pay
}
