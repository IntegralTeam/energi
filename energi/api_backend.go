// Copyright 2015 The go-ethereum Authors
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
	"context"
	"math/big"

	"github.com/IntegralTeam/energi/accounts"
	"github.com/IntegralTeam/energi/common"
	"github.com/IntegralTeam/energi/common/math"
	"github.com/IntegralTeam/energi/core"
	"github.com/IntegralTeam/energi/core/bloombits"
	"github.com/IntegralTeam/energi/core/state"
	"github.com/IntegralTeam/energi/core/types"
	"github.com/IntegralTeam/energi/core/vm"
	"github.com/IntegralTeam/energi/energi/downloader"
	"github.com/IntegralTeam/energi/energi/gasprice"
	"github.com/IntegralTeam/energi/energidb"
	"github.com/IntegralTeam/energi/event"
	"github.com/IntegralTeam/energi/params"
	"github.com/IntegralTeam/energi/rpc"
)

// EnergiAPIBackend implements energiapi.Backend for full nodes
type EnergiAPIBackend struct {
	energi *Energi
	gpo    *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *EnergiAPIBackend) ChainConfig() *params.ChainConfig {
	return b.energi.chainConfig
}

func (b *EnergiAPIBackend) CurrentBlock() *types.Block {
	return b.energi.blockchain.CurrentBlock()
}

func (b *EnergiAPIBackend) SetHead(number uint64) {
	b.energi.protocolManager.downloader.Cancel()
	b.energi.blockchain.SetHead(number)
}

func (b *EnergiAPIBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.energi.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.energi.blockchain.CurrentBlock().Header(), nil
	}
	return b.energi.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *EnergiAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.energi.blockchain.GetHeaderByHash(hash), nil
}

func (b *EnergiAPIBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.energi.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.energi.blockchain.CurrentBlock(), nil
	}
	return b.energi.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *EnergiAPIBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.energi.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.energi.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *EnergiAPIBackend) GetBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.energi.blockchain.GetBlockByHash(hash), nil
}

func (b *EnergiAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.energi.blockchain.GetReceiptsByHash(hash), nil
}

func (b *EnergiAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	receipts := b.energi.blockchain.GetReceiptsByHash(hash)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *EnergiAPIBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.energi.blockchain.GetTdByHash(blockHash)
}

func (b *EnergiAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.energi.BlockChain(), nil)
	return vm.NewEVM(context, state, b.energi.chainConfig, vmCfg), vmError, nil
}

func (b *EnergiAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.energi.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *EnergiAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.energi.BlockChain().SubscribeChainEvent(ch)
}

func (b *EnergiAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.energi.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *EnergiAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.energi.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *EnergiAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.energi.BlockChain().SubscribeLogsEvent(ch)
}

func (b *EnergiAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.energi.txPool.AddLocal(signedTx)
}

func (b *EnergiAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.energi.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *EnergiAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.energi.txPool.Get(hash)
}

func (b *EnergiAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.energi.txPool.State().GetNonce(addr), nil
}

func (b *EnergiAPIBackend) Stats() (pending int, queued int) {
	return b.energi.txPool.Stats()
}

func (b *EnergiAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.energi.TxPool().Content()
}

func (b *EnergiAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.energi.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *EnergiAPIBackend) Downloader() *downloader.Downloader {
	return b.energi.Downloader()
}

func (b *EnergiAPIBackend) ProtocolVersion() int {
	return b.energi.EnergiVersion()
}

func (b *EnergiAPIBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *EnergiAPIBackend) ChainDb() energidb.Database {
	return b.energi.ChainDb()
}

func (b *EnergiAPIBackend) EventMux() *event.TypeMux {
	return b.energi.EventMux()
}

func (b *EnergiAPIBackend) AccountManager() *accounts.Manager {
	return b.energi.AccountManager()
}

func (b *EnergiAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.energi.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *EnergiAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.energi.bloomRequests)
	}
}
