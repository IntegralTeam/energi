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
	"github.com/IntegralTeam/energi/common"
	"github.com/IntegralTeam/energi/rlp"
	"github.com/magiconair/properties/assert"
	"math/big"
	"reflect"
	"testing"
)

func TestDismissVote_Serialization(t *testing.T) {
	vote := DismissVote{}

	vote.Timestamp = 0xFFFFFFFFFFFF // 6 bytes
	assert.Equal(t, vote.Time().Unix(), int64(vote.Timestamp))
	vote.Auth.Sig = make([]byte, 65, 65)
	vote.Auth.Sig[0] = 5
	vote.Reason.Code = DissmissVote_NoHeartbeats
	vote.Reason.Description = "No one liked him"
	vote.CraAddressToDismiss = common.HexToAddress("93197b9019527e516b87317ebd065f240d972d22")
	vote.ExpirationBlockNumber = big.NewInt(1)

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	err := rlp.Encode(writer, &vote)
	assert.Equal(t, err, nil)
	writer.Flush()

	voteDecoded := DismissVote{}
	reader := bufio.NewReader(&b)
	s := rlp.NewStream(reader, uint64(len(b.Bytes())))
	err = s.Decode(&voteDecoded)
	assert.Equal(t, err, nil)

	assert.Equal(t, voteDecoded, vote)
}

// Check that different messages have different Hash and GetDataToSign
func TestDismissVote_GetDataToSign_Hash(t *testing.T) {
	vote := DismissVote{}

	vote.Timestamp = 0xFFFFFFFFFFFF // 6 bytes
	vote.Auth.Sig = make([]byte, 65, 65)
	vote.Auth.Sig[0] = 5
	vote.Reason.Code = DissmissVote_NoHeartbeats
	vote.Reason.Description = "No one liked him"
	vote.CraAddressToDismiss = common.HexToAddress("93197b9019527e516b87317ebd065f240d972d22")
	vote.ExpirationBlockNumber = big.NewInt(1)

	vote2 := vote
	vote2.Timestamp = 1

	vote3 := vote
	vote3.Auth.Sig = make([]byte, 65, 65)
	vote3.Auth.Sig[0] = 6

	vote4 := vote
	vote4.Reason.Code = DissmissVote_Another

	vote5 := vote
	vote5.Reason.Description = ""

	vote6 := vote
	vote6.CraAddressToDismiss = common.HexToAddress("c192752af76b34ea21fbf71b76a872b1282d02fd")

	vote7 := vote
	vote7.ExpirationBlockNumber = big.NewInt(2)

	assert.Equal(t, vote.Hash().String() != vote2.Hash().String(), true)
	assert.Equal(t, vote.Hash().String() != vote3.Hash().String(), true)
	assert.Equal(t, vote.Hash().String() != vote4.Hash().String(), true)
	assert.Equal(t, vote.Hash().String() != vote5.Hash().String(), true)
	assert.Equal(t, vote.Hash().String() != vote6.Hash().String(), true)
	assert.Equal(t, vote.Hash().String() != vote7.Hash().String(), true)

	assert.Equal(t, reflect.DeepEqual(vote.GetDataToSign(), vote2.GetDataToSign()), false)
	assert.Equal(t, reflect.DeepEqual(vote.GetDataToSign(), vote3.GetDataToSign()), true)
	assert.Equal(t, reflect.DeepEqual(vote.GetDataToSign(), vote4.GetDataToSign()), false)
	assert.Equal(t, reflect.DeepEqual(vote.GetDataToSign(), vote5.GetDataToSign()), false)
	assert.Equal(t, reflect.DeepEqual(vote.GetDataToSign(), vote6.GetDataToSign()), false)
	assert.Equal(t, reflect.DeepEqual(vote.GetDataToSign(), vote7.GetDataToSign()), false)
}
