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
	"errors"
	"fmt"
	"github.com/IntegralTeam/energi/accounts"
	"github.com/IntegralTeam/energi/common"
	"github.com/IntegralTeam/energi/common/hexutil"
	"github.com/IntegralTeam/energi/crypto"
	"github.com/IntegralTeam/energi/rlp"
	"github.com/IntegralTeam/energi/signer/core"
	"io"
	"math/big"
)

//go:generate gencodec -type Auth -field-override authMarshaling -out gen_auth_json.go

type Auth struct {
	Sig []byte `json:"sig" gencodec:"required"`
}

type authMarshaling struct {
	Sig hexutil.Bytes
}

// EncodeRLP implements rlp.Encoder
func (auth *Auth) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &auth.Sig)
}

// DecodeRLP implements rlp.Decoder
func (auth *Auth) DecodeRLP(s *rlp.Stream) error {
	return s.Decode(&auth.Sig)
}

// Retrieve V, R, S from raw signature
func (auth *Auth) GetSignatureValues() (*big.Int, *big.Int, *big.Int, error) {
	if len(auth.Sig) != 65 {
		return nil, nil, nil, errors.New(fmt.Sprintf("wrong size for signature: got %d, want 65", len(auth.Sig)))
	}

	r := new(big.Int).SetBytes(auth.Sig[:32])
	s := new(big.Int).SetBytes(auth.Sig[32:64])
	v := new(big.Int).SetBytes([]byte{auth.Sig[64] + 27})

	return v, r, s, nil
}

// Calculate signer address
func (auth *Auth) GetSignatureAddress(message []byte) (common.Address, error) {
	if len(auth.Sig) != 65 {
		return common.Address{}, errors.New(fmt.Sprintf("wrong size for signature: got %d, want 65", len(auth.Sig)))
	}

	hash, _ := core.SignHash(message)
	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(hash, auth.Sig)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	return signer, nil
}

func (auth *Auth) Sign(message []byte, wallet accounts.Wallet, account accounts.Account, passphrase string) error {
	err := wallet.Open(passphrase)
	if err != nil {
		return err
	}
	hash, _ := core.SignHash(message)
	auth.Sig, err = wallet.SignHashWithPassphrase(account, passphrase, hash)
	return err
}
