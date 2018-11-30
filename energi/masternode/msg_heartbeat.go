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
	"time"
)

//go:generate gencodec -type Heartbeat -out gen_heartbeat_json.go

type Heartbeat struct {
	Timestamp uint64 `json:"time" gencodec:"required"`
	Auth      Auth   `json:"auth" gencodec:"required"`
}

func (h *Heartbeat) Hash() common.Hash {
	// Serialize using RLP
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	rlp.Encode(writer, &h)
	writer.Flush()

	// Calc hash
	var hash common.Hash
	copy(hash[:], crypto.Keccak256(b.Bytes())[:32])

	return hash
}

func (h *Heartbeat) Time() time.Time {
	return time.Unix(int64(h.Timestamp), 0)
}

// Build data to be used for signing
func (h *Heartbeat) GetDataToSign() []byte {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	// heartbeat header
	writer.Write([]byte{0x90, 0x6c, 0x56, 0x1b, 0x1b, 0x1e, 0x76, 0xed})
	// heartbeat body
	timestampB := make([]byte, 8, 8)
	binary.LittleEndian.PutUint64(timestampB, h.Timestamp)
	writer.Write(timestampB)

	writer.Flush()
	return b.Bytes()
}
