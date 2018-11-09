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

package energiclient

import "github.com/IntegralTeam/energi"

// Verify that Client implements the energi interfaces.
var (
	_ = energi.ChainReader(&Client{})
	_ = energi.TransactionReader(&Client{})
	_ = energi.ChainStateReader(&Client{})
	_ = energi.ChainSyncReader(&Client{})
	_ = energi.ContractCaller(&Client{})
	_ = energi.GasEstimator(&Client{})
	_ = energi.GasPricer(&Client{})
	_ = energi.LogFilterer(&Client{})
	_ = energi.PendingStateReader(&Client{})
	// _ = ethereum.PendingStateEventer(&Client{})
	_ = energi.PendingContractCaller(&Client{})
)
