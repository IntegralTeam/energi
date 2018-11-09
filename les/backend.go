// Copyright 2016 The go-ethereum Authors
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

// Package les implements the Light Energi Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/IntegralTeam/energi/accounts"
	"github.com/IntegralTeam/energi/common"
	"github.com/IntegralTeam/energi/common/hexutil"
	"github.com/IntegralTeam/energi/consensus"
	"github.com/IntegralTeam/energi/core"
	"github.com/IntegralTeam/energi/core/bloombits"
	"github.com/IntegralTeam/energi/core/rawdb"
	"github.com/IntegralTeam/energi/core/types"
	"github.com/IntegralTeam/energi/energi"
	"github.com/IntegralTeam/energi/energi/downloader"
	"github.com/IntegralTeam/energi/energi/filters"
	"github.com/IntegralTeam/energi/energi/gasprice"
	"github.com/IntegralTeam/energi/event"
	"github.com/IntegralTeam/energi/internal/energiapi"
	"github.com/IntegralTeam/energi/light"
	"github.com/IntegralTeam/energi/log"
	"github.com/IntegralTeam/energi/node"
	"github.com/IntegralTeam/energi/p2p"
	"github.com/IntegralTeam/energi/p2p/discv5"
	"github.com/IntegralTeam/energi/params"
	rpc "github.com/IntegralTeam/energi/rpc"
)

type LightEnergi struct {
	lesCommons

	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool

	// Handlers
	peers      *peerSet
	txPool     *light.TxPool
	blockchain *light.LightChain
	serverPool *serverPool
	reqDist    *requestDistributor
	retriever  *retrieveManager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *energiapi.PublicNetAPI

	wg sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *energi.Config) (*LightEnergi, error) {
	chainDb, err := energi.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	lenergi := &LightEnergi{
		lesCommons: lesCommons{
			chainDb: chainDb,
			config:  config,
			iConfig: light.DefaultClientIndexerConfig,
		},
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		peers:          peers,
		reqDist:        newRequestDistributor(peers, quitSync),
		accountManager: ctx.AccountManager,
		engine:         energi.CreateConsensusEngine(ctx, chainConfig, *config, chainDb),
		shutdownChan:   make(chan bool),
		networkId:      config.NetworkId,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   energi.NewBloomIndexer(chainDb, params.BloomBitsBlocksClient, params.HelperTrieConfirmations),
	}

	lenergi.relay = NewLesTxRelay(peers, lenergi.reqDist)
	lenergi.serverPool = newServerPool(chainDb, quitSync, &lenergi.wg)
	lenergi.retriever = newRetrieveManager(peers, lenergi.reqDist, lenergi.serverPool)

	lenergi.odr = NewLesOdr(chainDb, light.DefaultClientIndexerConfig, lenergi.retriever)
	lenergi.chtIndexer = light.NewChtIndexer(chainDb, lenergi.odr, params.CHTFrequencyClient, params.HelperTrieConfirmations)
	lenergi.bloomTrieIndexer = light.NewBloomTrieIndexer(chainDb, lenergi.odr, params.BloomBitsBlocksClient, params.BloomTrieFrequency)
	lenergi.odr.SetIndexers(lenergi.chtIndexer, lenergi.bloomTrieIndexer, lenergi.bloomIndexer)

	// Note: NewLightChain adds the trusted checkpoint so it needs an ODR with
	// indexers already set but not started yet
	if lenergi.blockchain, err = light.NewLightChain(lenergi.odr, lenergi.chainConfig, lenergi.engine); err != nil {
		return nil, err
	}
	// Note: AddChildIndexer starts the update process for the child
	lenergi.bloomIndexer.AddChildIndexer(lenergi.bloomTrieIndexer)
	lenergi.chtIndexer.Start(lenergi.blockchain)
	lenergi.bloomIndexer.Start(lenergi.blockchain)

	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		lenergi.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	lenergi.txPool = light.NewTxPool(lenergi.chainConfig, lenergi.blockchain, lenergi.relay)
	if lenergi.protocolManager, err = NewProtocolManager(lenergi.chainConfig, light.DefaultClientIndexerConfig, true, config.NetworkId, lenergi.eventMux, lenergi.engine, lenergi.peers, lenergi.blockchain, nil, chainDb, lenergi.odr, lenergi.relay, lenergi.serverPool, quitSync, &lenergi.wg); err != nil {
		return nil, err
	}
	lenergi.ApiBackend = &LesApiBackend{lenergi, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.MinerGasPrice
	}
	lenergi.ApiBackend.gpo = gasprice.NewOracle(lenergi.ApiBackend, gpoParams)
	return lenergi, nil
}

func lesTopic(genesisHash common.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case lpv1:
		name = "LES"
	case lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// Energibase is the address that mining rewards will be send to
func (s *LightDummyAPI) Energibase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Coinbase is the address that mining rewards will be send to (alias for Energibase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the energi package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (energi *LightEnergi) APIs() []rpc.API {
	return append(energiapi.GetAPIs(energi.ApiBackend), []rpc.API{
		{
			Namespace: "energi",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "energi",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(energi.protocolManager.downloader, energi.eventMux),
			Public:    true,
		}, {
			Namespace: "energi",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(energi.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   energi.netRPCService,
			Public:    true,
		},
	}...)
}

func (energi *LightEnergi) ResetWithGenesisBlock(gb *types.Block) {
	energi.blockchain.ResetWithGenesisBlock(gb)
}

func (energi *LightEnergi) BlockChain() *light.LightChain { return energi.blockchain }
func (energi *LightEnergi) TxPool() *light.TxPool         { return energi.txPool }
func (energi *LightEnergi) Engine() consensus.Engine      { return energi.engine }
func (energi *LightEnergi) LesVersion() int               { return int(ClientProtocolVersions[0]) }
func (energi *LightEnergi) Downloader() *downloader.Downloader {
	return energi.protocolManager.downloader
}
func (energi *LightEnergi) EventMux() *event.TypeMux { return energi.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (energi *LightEnergi) Protocols() []p2p.Protocol {
	return energi.makeProtocols(ClientProtocolVersions)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Energi protocol implementation.
func (energi *LightEnergi) Start(srvr *p2p.Server) error {
	log.Warn("Light client mode is an experimental feature")
	energi.startBloomHandlers(params.BloomBitsBlocksClient)
	energi.netRPCService = energiapi.NewPublicNetAPI(srvr, energi.networkId)
	// clients are searching for the first advertised protocol in the list
	protocolVersion := AdvertiseProtocolVersions[0]
	energi.serverPool.start(srvr, lesTopic(energi.blockchain.Genesis().Hash(), protocolVersion))
	energi.protocolManager.Start(energi.config.LightPeers)
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Energi protocol.
func (energi *LightEnergi) Stop() error {
	energi.odr.Stop()
	energi.bloomIndexer.Close()
	energi.chtIndexer.Close()
	energi.blockchain.Stop()
	energi.protocolManager.Stop()
	energi.txPool.Stop()
	energi.engine.Close()

	energi.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	energi.chainDb.Close()
	close(energi.shutdownChan)

	return nil
}
