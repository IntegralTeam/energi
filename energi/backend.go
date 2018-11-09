// Copyright 2014 The go-ethereum Authors
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

// Package energi implements the Energi protocol.
package energi

import (
	"errors"
	"fmt"
	"github.com/IntegralTeam/energi/consensus/energihash"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/IntegralTeam/energi/accounts"
	"github.com/IntegralTeam/energi/common"
	"github.com/IntegralTeam/energi/common/hexutil"
	"github.com/IntegralTeam/energi/consensus"
	"github.com/IntegralTeam/energi/consensus/clique"
	"github.com/IntegralTeam/energi/consensus/ethash"
	"github.com/IntegralTeam/energi/core"
	"github.com/IntegralTeam/energi/core/bloombits"
	"github.com/IntegralTeam/energi/core/rawdb"
	"github.com/IntegralTeam/energi/core/types"
	"github.com/IntegralTeam/energi/core/vm"
	"github.com/IntegralTeam/energi/energi/downloader"
	"github.com/IntegralTeam/energi/energi/filters"
	"github.com/IntegralTeam/energi/energi/gasprice"
	"github.com/IntegralTeam/energi/energidb"
	"github.com/IntegralTeam/energi/event"
	"github.com/IntegralTeam/energi/internal/energiapi"
	"github.com/IntegralTeam/energi/log"
	"github.com/IntegralTeam/energi/miner"
	"github.com/IntegralTeam/energi/node"
	"github.com/IntegralTeam/energi/p2p"
	"github.com/IntegralTeam/energi/params"
	"github.com/IntegralTeam/energi/rlp"
	"github.com/IntegralTeam/energi/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// Energi implements the Energi full node service.
type Energi struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the Energi

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb energidb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	APIBackend *EnergiAPIBackend

	miner      *miner.Miner
	gasPrice   *big.Int
	energibase common.Address

	networkID     uint64
	netRPCService *energiapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and energibase)
}

func (energi *Energi) AddLesServer(ls LesServer) {
	energi.lesServer = ls
	ls.SetBloomBitsIndexer(energi.bloomIndexer)
}

// New creates a new Energi object (including the
// initialisation of the common Energi object)
func New(ctx *node.ServiceContext, config *Config) (*Energi, error) {
	// Ensure configuration values are compatible and sane
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run energi.Energi in light sync mode, use les.LightEthereum")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	if config.MinerGasPrice == nil || config.MinerGasPrice.Cmp(common.Big0) <= 0 {
		log.Warn("Sanitizing invalid miner gas price", "provided", config.MinerGasPrice, "updated", DefaultConfig.MinerGasPrice)
		config.MinerGasPrice = new(big.Int).Set(DefaultConfig.MinerGasPrice)
	}
	// Assemble the Energi object
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	energi := &Energi{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, chainConfig, *config, chainDb),
		shutdownChan:   make(chan bool),
		networkID:      config.NetworkId,
		gasPrice:       config.MinerGasPrice,
		energibase:     config.Energibase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks, params.BloomConfirms),
	}

	log.Info("Initialising Energi protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := rawdb.ReadDatabaseVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d).\n", bcVersion, core.BlockChainVersion)
		}
		rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
	}
	var (
		vmConfig = vm.Config{
			EnablePreimageRecording: config.EnablePreimageRecording,
			EWASMInterpreter:        config.EWASMInterpreter,
			EVMInterpreter:          config.EVMInterpreter,
		}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieNodeLimit: config.TrieCache, TrieTimeLimit: config.TrieTimeout}
	)
	energi.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, energi.chainConfig, energi.engine, vmConfig, energi.shouldPreserve)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		energi.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	energi.bloomIndexer.Start(energi.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	energi.txPool = core.NewTxPool(config.TxPool, energi.chainConfig, energi.blockchain)

	if energi.protocolManager, err = NewProtocolManager(energi.chainConfig, config.SyncMode, config.NetworkId, energi.eventMux, energi.txPool, energi.engine, energi.blockchain, chainDb); err != nil {
		return nil, err
	}

	energi.miner = miner.New(energi, energi.chainConfig, energi.EventMux(), energi.engine, config.MinerRecommit, config.MinerGasFloor, config.MinerGasCeil, energi.isLocalBlock)
	energi.miner.SetExtra(makeExtraData(config.MinerExtraData))

	energi.APIBackend = &EnergiAPIBackend{energi, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.MinerGasPrice
	}
	energi.APIBackend.gpo = gasprice.NewOracle(energi.APIBackend, gpoParams)

	return energi, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"energi",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (energidb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*energidb.LDBDatabase); ok {
		db.Meter("energi/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Energi service
func CreateConsensusEngine(ctx *node.ServiceContext, chainConfig *params.ChainConfig, config Config, db energidb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// If we use energihash instead of ethash, set it up
	if chainConfig.Energihash != nil {
		switch config.Energihash.PowMode {
		case energihash.ModeFake:
			log.Warn("Energihash used in fake mode")
			return energihash.NewFaker()
		case energihash.ModeTest:
			log.Warn("Energihash used in test mode")
			return energihash.NewTester(nil, config.MinerNoverify)
		case energihash.ModeShared:
			log.Warn("Energihash used in shared mode")
			return ethash.NewShared()
		default:
			return energihash.New(energihash.Config{
				CacheDir:       ctx.ResolvePath(config.Energihash.CacheDir),
				CachesInMem:    config.Energihash.CachesInMem,
				CachesOnDisk:   config.Energihash.CachesOnDisk,
				DatasetDir:     config.Energihash.DatasetDir,
				DatasetsInMem:  config.Energihash.DatasetsInMem,
				DatasetsOnDisk: config.Energihash.DatasetsOnDisk,
			}, config.MinerNotify, config.MinerNoverify)
		}
	}
	// Otherwise assume ethash proof-of-work
	switch config.Ethash.PowMode {
	case ethash.ModeFake:
		log.Warn("Ethash used in fake mode")
		return ethash.NewFaker()
	case ethash.ModeTest:
		log.Warn("Ethash used in test mode")
		return ethash.NewTester(nil, config.MinerNoverify)
	case ethash.ModeShared:
		log.Warn("Ethash used in shared mode")
		return ethash.NewShared()
	default:
		engine := ethash.New(ethash.Config{
			CacheDir:       ctx.ResolvePath(config.Ethash.CacheDir),
			CachesInMem:    config.Ethash.CachesInMem,
			CachesOnDisk:   config.Ethash.CachesOnDisk,
			DatasetDir:     config.Ethash.DatasetDir,
			DatasetsInMem:  config.Ethash.DatasetsInMem,
			DatasetsOnDisk: config.Ethash.DatasetsOnDisk,
		}, config.MinerNotify, config.MinerNoverify)
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs return the collection of RPC services the energi package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (energi *Energi) APIs() []rpc.API {
	apis := energiapi.GetAPIs(energi.APIBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, energi.engine.APIs(energi.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "energi",
			Version:   "1.0",
			Service:   NewPublicEnergiAPI(energi),
			Public:    true,
		}, {
			Namespace: "energi",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(energi),
			Public:    true,
		}, {
			Namespace: "energi",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(energi.protocolManager.downloader, energi.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(energi),
			Public:    false,
		}, {
			Namespace: "energi",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(energi.APIBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(energi),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(energi),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(energi.chainConfig, energi),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   energi.netRPCService,
			Public:    true,
		},
	}...)
}

func (energi *Energi) ResetWithGenesisBlock(gb *types.Block) {
	energi.blockchain.ResetWithGenesisBlock(gb)
}

func (energi *Energi) Energibase() (eb common.Address, err error) {
	energi.lock.RLock()
	energibase := energi.energibase
	energi.lock.RUnlock()

	if energibase != (common.Address{}) {
		return energibase, nil
	}
	if wallets := energi.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			energibase := accounts[0].Address

			energi.lock.Lock()
			energi.energibase = energibase
			energi.lock.Unlock()

			log.Info("Energibase automatically configured", "address", energibase)
			return energibase, nil
		}
	}
	return common.Address{}, fmt.Errorf("energibase must be explicitly specified")
}

// isLocalBlock checks whether the specified block is mined
// by local miner accounts.
//
// We regard two types of accounts as local miner account: energibase
// and accounts specified via `txpool.locals` flag.
func (energi *Energi) isLocalBlock(block *types.Block) bool {
	author, err := energi.engine.Author(block.Header())
	if err != nil {
		log.Warn("Failed to retrieve block author", "number", block.NumberU64(), "hash", block.Hash(), "err", err)
		return false
	}
	// Check whether the given address is energibase.
	energi.lock.RLock()
	energibase := energi.energibase
	energi.lock.RUnlock()
	if author == energibase {
		return true
	}
	// Check whether the given address is specified by `txpool.local`
	// CLI flag.
	for _, account := range energi.config.TxPool.Locals {
		if account == author {
			return true
		}
	}
	return false
}

// shouldPreserve checks whether we should preserve the given block
// during the chain reorg depending on whether the author of block
// is a local account.
func (energi *Energi) shouldPreserve(block *types.Block) bool {
	// The reason we need to disable the self-reorg preserving for clique
	// is it can be probable to introduce a deadlock.
	//
	// e.g. If there are 7 available signers
	//
	// r1   A
	// r2     B
	// r3       C
	// r4         D
	// r5   A      [X] F G
	// r6    [X]
	//
	// In the round5, the inturn signer E is offline, so the worst case
	// is A, F and G sign the block of round5 and reject the block of opponents
	// and in the round6, the last available signer B is offline, the whole
	// network is stuck.
	if _, ok := energi.engine.(*clique.Clique); ok {
		return false
	}
	return energi.isLocalBlock(block)
}

// SetEnergibase sets the mining reward address.
func (energi *Energi) SetEnergibase(energibase common.Address) {
	energi.lock.Lock()
	energi.energibase = energibase
	energi.lock.Unlock()

	energi.miner.SetEnergibase(energibase)
}

// StartMining starts the miner with the given number of CPU threads. If mining
// is already running, this method adjust the number of threads allowed to use
// and updates the minimum price required by the transaction pool.
func (energi *Energi) StartMining(threads int) error {
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := energi.engine.(threaded); ok {
		log.Info("Updated mining threads", "threads", threads)
		if threads == 0 {
			threads = -1 // Disable the miner from within
		}
		th.SetThreads(threads)
	}
	// If the miner was not running, initialize it
	if !energi.IsMining() {
		// Propagate the initial price point to the transaction pool
		energi.lock.RLock()
		price := energi.gasPrice
		energi.lock.RUnlock()
		energi.txPool.SetGasPrice(price)

		// Configure the local mining address
		eb, err := energi.Energibase()
		if err != nil {
			log.Error("Cannot start mining without energibase", "err", err)
			return fmt.Errorf("energibase missing: %v", err)
		}
		if clique, ok := energi.engine.(*clique.Clique); ok {
			wallet, err := energi.accountManager.Find(accounts.Account{Address: eb})
			if wallet == nil || err != nil {
				log.Error("Energibase account unavailable locally", "err", err)
				return fmt.Errorf("signer missing: %v", err)
			}
			clique.Authorize(eb, wallet.SignHash)
		}
		// If mining is started, we can disable the transaction rejection mechanism
		// introduced to speed sync times.
		atomic.StoreUint32(&energi.protocolManager.acceptTxs, 1)

		go energi.miner.Start(eb)
	}
	return nil
}

// StopMining terminates the miner, both at the consensus engine level as well as
// at the block creation level.
func (energi *Energi) StopMining() {
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := energi.engine.(threaded); ok {
		th.SetThreads(-1)
	}
	// Stop the block creating itself
	energi.miner.Stop()
}

func (energi *Energi) IsMining() bool      { return energi.miner.Mining() }
func (energi *Energi) Miner() *miner.Miner { return energi.miner }

func (energi *Energi) AccountManager() *accounts.Manager  { return energi.accountManager }
func (energi *Energi) BlockChain() *core.BlockChain       { return energi.blockchain }
func (energi *Energi) TxPool() *core.TxPool               { return energi.txPool }
func (energi *Energi) EventMux() *event.TypeMux           { return energi.eventMux }
func (energi *Energi) Engine() consensus.Engine           { return energi.engine }
func (energi *Energi) ChainDb() energidb.Database         { return energi.chainDb }
func (energi *Energi) IsListening() bool                  { return true } // Always listening
func (energi *Energi) EnergiVersion() int                 { return int(energi.protocolManager.SubProtocols[0].Version) }
func (energi *Energi) NetVersion() uint64                 { return energi.networkID }
func (energi *Energi) Downloader() *downloader.Downloader { return energi.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (energi *Energi) Protocols() []p2p.Protocol {
	if energi.lesServer == nil {
		return energi.protocolManager.SubProtocols
	}
	return append(energi.protocolManager.SubProtocols, energi.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Energi protocol implementation.
func (energi *Energi) Start(server *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	energi.startBloomHandlers(params.BloomBitsBlocks)

	// Start the RPC service
	energi.netRPCService = energiapi.NewPublicNetAPI(server, energi.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := server.MaxPeers
	if energi.config.LightServ > 0 {
		if energi.config.LightPeers >= server.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", energi.config.LightPeers, server.MaxPeers)
		}
		maxPeers -= energi.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	energi.protocolManager.Start(maxPeers)
	if energi.lesServer != nil {
		energi.lesServer.Start(server)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Energi protocol.
func (energi *Energi) Stop() error {
	energi.bloomIndexer.Close()
	energi.blockchain.Stop()
	energi.engine.Close()
	energi.protocolManager.Stop()
	if energi.lesServer != nil {
		energi.lesServer.Stop()
	}
	energi.txPool.Stop()
	energi.miner.Stop()
	energi.eventMux.Stop()

	energi.chainDb.Close()
	close(energi.shutdownChan)
	return nil
}
