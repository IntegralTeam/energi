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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/IntegralTeam/energi/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("energi/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("energi/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("energi/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("energi/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("energi/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("energi/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("energi/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("energi/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("energi/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("energi/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("energi/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("energi/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("energi/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("energi/downloader/states/drop", nil)
)
