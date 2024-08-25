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

// Pallert
const (
	_FILEBANK = "FileBank"
	_SYSTEM   = "System"
	_CACHER   = "Cacher"
)

// Chain state
const (
	// System
	_SYSTEM_ACCOUNT = "Account"
	_SYSTEM_EVENTS  = "Events"
	// FileMap
	_FILEMAP_FILEMETA = "File"
	// Miner
	_CACHER_CACHER = "Cachers"
)

// Extrinsics
const (
	CACHER_REGISTER = "Cacher.register"
	CACHER_UPDATE   = "Cacher.update"
	CACHER_LOGOUT   = "Cacher.logout"
	CACHER_PAY      = "Cacher.pay"
)

const (
	FILE_STATE_ACTIVE  = "active"
	FILE_STATE_PENDING = "pending"
)

const (
	MINER_STATE_POSITIVE = "positive"
	MINER_STATE_FROZEN   = "frozen"
	MINER_STATE_EXIT     = "exit"
)
