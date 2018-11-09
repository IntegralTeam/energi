// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Copyright 2018 The energi Authors
// This file is part of the energi library.
//
// The energi library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The energi library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the energi library. If not, see <http://www.gnu.org/licenses/>.

package energi

import (
	"github.com/IntegralTeam/energi/common"
	"github.com/IntegralTeam/energi/common/hexutil"
	"github.com/IntegralTeam/energi/consensus/energihash"
	"github.com/IntegralTeam/energi/consensus/ethash"
	"github.com/IntegralTeam/energi/core"
	"github.com/IntegralTeam/energi/energi/downloader"
	"github.com/IntegralTeam/energi/energi/gasprice"
	"github.com/IntegralTeam/energi/params"
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"
)

// DefaultConfig contains default settings for use on the Energi main net.
var DefaultConfig = Config{
	SyncMode: downloader.FastSync,
	Energihash: energihash.Config{
		CacheDir:       "energihash",
		CachesInMem:    2,
		CachesOnDisk:   3,
		DatasetsInMem:  1,
		DatasetsOnDisk: 2,
	},
	NetworkId:     1,
	LightPeers:    100,
	DatabaseCache: 768,
	TrieCache:     256,
	TrieTimeout:   60 * time.Minute,
	MinerGasFloor: 8000000,
	MinerGasCeil:  8000000,
	MinerGasPrice: big.NewInt(params.GWei),
	MinerRecommit: 3 * time.Second,

	TxPool: core.DefaultTxPoolConfig,
	GPO: gasprice.Config{
		Blocks:     20,
		Percentile: 60,
	},
}

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
	if runtime.GOOS == "windows" {
		DefaultConfig.Energihash.DatasetDir = filepath.Join(home, "AppData", "Energihash")
	} else {
		DefaultConfig.Energihash.DatasetDir = filepath.Join(home, ".energihash")
	}
}

//go:generate gencodec -type Config -field-override configMarshaling -formats toml -out gen_config.go

type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the Energi main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Protocol options
	NetworkId uint64 // Network ID to use for selecting peers to connect to
	SyncMode  downloader.SyncMode
	NoPruning bool

	// Light client options
	LightServ  int `toml:",omitempty"` // Maximum percentage of time allowed for serving LES requests
	LightPeers int `toml:",omitempty"` // Maximum number of LES client peers

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int
	TrieCache          int
	TrieTimeout        time.Duration

	// Mining-related options
	Energibase     common.Address `toml:",omitempty"`
	MinerNotify    []string       `toml:",omitempty"`
	MinerExtraData []byte         `toml:",omitempty"`
	MinerGasFloor  uint64
	MinerGasCeil   uint64
	MinerGasPrice  *big.Int
	MinerRecommit  time.Duration
	MinerNoverify  bool

	// Ethash options
	Ethash ethash.Config

	// Energihash options
	Energihash energihash.Config

	// Transaction pool options
	TxPool core.TxPoolConfig

	// Gas Price Oracle options
	GPO gasprice.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options
	DocRoot string `toml:"-"`

	// Type of the EWASM interpreter ("" for detault)
	EWASMInterpreter string
	// Type of the EVM interpreter ("" for default)
	EVMInterpreter string
}

type configMarshaling struct {
	MinerExtraData hexutil.Bytes
}
