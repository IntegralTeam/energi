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

package mn_back

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"github.com/IntegralTeam/energi/common"
	"github.com/IntegralTeam/energi/crypto"
	"github.com/IntegralTeam/energi/rlp"
	"math/big"
	"time"
)

type DismissingReasonCode uint32

const (
	DissmissVote_NoHeartbeats = 0x01
	DissmissVote_Another      = 0xffffffff
)

type DismissingReason struct {
	Code        DismissingReasonCode `json:"code" gencodec:"required"`
	Description string               `json:"description" gencodec:"required"`
}

//go:generate gencodec -type DismissVote -out gen_dismiss_vote_json.go

type DismissVote struct {
	CraAddressToDismiss   common.Address   `json:"craAddressToDismiss" gencodec:"required"`
	ExpirationBlockNumber *big.Int         `json:"expirationBlockNumber" gencodec:"required"`
	Reason                DismissingReason `json:"reason" gencodec:"required"`
	Timestamp             uint64           `json:"time" gencodec:"required"`
	Auth                  Auth             `json:"auth" gencodec:"required"`
}

func (v *DismissVote) Hash() common.Hash {
	// Serialize using RLP
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	rlp.Encode(writer, &v)
	writer.Flush()

	// Calc hash
	var hash common.Hash
	copy(hash[:], crypto.Keccak256(b.Bytes())[:32])

	return hash
}

func (v *DismissVote) Time() time.Time {
	return time.Unix(int64(v.Timestamp), 0)
}

// Build data to be used for signing
func (v *DismissVote) GetDataToSign() []byte {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	// DismissVote header
	writer.Write([]byte{0xf8, 0x79, 0x21, 0xf1, 0x9a, 0x83, 0xf3, 0x9d})

	// DismissVote body
	writer.Write(v.CraAddressToDismiss.Bytes())

	// serialize as 8 bytes to make it simpler to reproduce this code in other langs
	expirationBlockNumberB := make([]byte, 8, 8)
	binary.LittleEndian.PutUint64(expirationBlockNumberB, v.ExpirationBlockNumber.Uint64())
	writer.Write(expirationBlockNumberB)

	reasonCodeB := make([]byte, 4, 4)
	binary.LittleEndian.PutUint32(reasonCodeB, uint32(v.Reason.Code))
	writer.Write(reasonCodeB)
	writer.WriteString(v.Reason.Description)

	timestampB := make([]byte, 8, 8)
	binary.LittleEndian.PutUint64(timestampB, v.Timestamp)
	writer.Write(timestampB)

	writer.Flush()
	return b.Bytes()
}
