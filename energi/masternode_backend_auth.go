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
	"github.com/IntegralTeam/energi/accounts"
	"github.com/IntegralTeam/energi/common"
	"github.com/IntegralTeam/energi/consensus/masternode"
	"github.com/IntegralTeam/energi/energi/masternode"
	"github.com/IntegralTeam/energi/log"
	"github.com/pkg/errors"
	"math/big"
)

// backend.lock is supposed to be locked
// Called by: masternode
func AuthenticateMessage(dataToSign []byte, auth *mn_back.Auth, block_number *big.Int) (common.Address, *ErrCode) {
	craAddress, err := auth.GetSignatureAddress(dataToSign)
	if err != nil {
		return common.Address{}, &ErrCode{ErrAuthWrongSignature}
	}

	_, ok := mn.GetActiveMasternodesMap(block_number)[craAddress]
	if !ok {
		return craAddress, &ErrCode{ErrAuthMasternodeNotFound}
	}
	return craAddress, nil
}

// backend.lock is supposed to be locked
func (backend *MasternodeBackend) signMessage(dataToSign []byte, auth *mn_back.Auth) error {
	account := accounts.Account{
		Address: backend.Config.CraAddress,
		URL:     accounts.URL{},
	}
	wallet, err := backend.accountManager.Find(account)
	if err != nil {
		log.Error("Unable to find a wallet to sign a masternode message", "address", backend.Config.CraAddress.String())
		return errors.Wrap(err, "Unable to find a wallet to sign a masternode message")
	}

	err = auth.Sign(dataToSign, wallet, account, backend.Config.Passphrase)
	if err != nil {
		log.Error("Unable to sign a masternode message", "err", err)
		return errors.Wrap(err, "Unable to sign a masternode message")
	}

	return nil
}

// backend.lock is supposed to be locked
func (backend *MasternodeBackend) amIActiveMasternode() bool {
	block_number := backend.protocolManager.blockchain.CurrentBlock().Header().Number

	// Ensure we're an active masternode
	_, ok := mn.GetMasternodesMap()[backend.Config.CraAddress]
	if !ok {
		log.Error("Masternode isn't announced", "block", block_number.String(), "address", backend.Config.CraAddress.String())
		return false
	}

	_, ok = mn.GetActiveMasternodesMap(block_number)[backend.Config.CraAddress]
	if !ok {
		log.Warn("Masternode isn't activated", "block", block_number.String(), "address", backend.Config.CraAddress.String())
		return false
	}
	return true
}
