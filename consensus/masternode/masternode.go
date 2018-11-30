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

package mn

import (
	"fmt"
	"github.com/IntegralTeam/energi/common"
	"github.com/IntegralTeam/energi/p2p/discv5"
	"github.com/IntegralTeam/energi/params"
	"math/big"
)

// Represents Masternode. This state is stored inside masternodes smart contract.
type Masternode struct {
	Alias string // human-readable name

	// net addresses
	NodeAddressIpV4 *discv5.NodeAddress
	NodeAddressIpV6 *discv5.NodeAddress // Optional network address

	CollateralAmount        *big.Int
	CraAddress              common.Address // CRA (Collateral/Reward/Authentication) address. The address from which the collateral was sent
	AnnouncementBlockNumber *big.Int       // The block in which the tx-Announce was included
	ActivationBlockNumber   *big.Int       // Formula: <Announcement block number> + max(round_up(<whole collateral> / <MinCollateral>), 100)
}

// Minimum masternode collateral
var MinCollateral = new(big.Int).Mul(big.NewInt(10000), params.Energi_bn) // 10000 NRG

// Return only activated masternodes
func FilterNotActiveMasternodes(masternodes []*Masternode, block_number *big.Int) []*Masternode {
	masternodesFiltered := make([]*Masternode, 0, len(masternodes))

	for _, masternode := range masternodes {
		if block_number.Cmp(masternode.ActivationBlockNumber) >= 0 {
			masternodesFiltered = append(masternodesFiltered, masternode)
		}
	}
	return masternodesFiltered
}

func GetMasternodes() []*Masternode {
	masternodes := make([]*Masternode, 3, 3)

	for i, _ := range masternodes {
		masternodes[i] = &Masternode{
			Alias:                   fmt.Sprintf("MN%d", i),
			NodeAddressIpV4:         nil,
			NodeAddressIpV6:         nil,
			CollateralAmount:        new(big.Int),
			CraAddress:              common.Address{},
			AnnouncementBlockNumber: new(big.Int),
			ActivationBlockNumber:   new(big.Int),
		}
	}
	masternodes[0].CollateralAmount = new(big.Int).Mul(big.NewInt(10000), params.Energi_bn)
	masternodes[0].AnnouncementBlockNumber = big.NewInt(0)
	masternodes[0].ActivationBlockNumber = big.NewInt(4)
	masternodes[0].CraAddress = common.HexToAddress("93197b9019527e516b87317ebd065f240d972d22") // pass is 1

	masternodes[1].CollateralAmount = new(big.Int).Mul(big.NewInt(10000), params.Energi_bn)
	masternodes[1].AnnouncementBlockNumber = big.NewInt(10)
	masternodes[1].ActivationBlockNumber = big.NewInt(14)
	masternodes[1].CraAddress = common.HexToAddress("c192752af76b34ea21fbf71b76a872b1282d02fd") // pass is 2

	masternodes[2].CollateralAmount = new(big.Int).Mul(big.NewInt(10000), params.Energi_bn)
	masternodes[2].AnnouncementBlockNumber = big.NewInt(20)
	masternodes[2].ActivationBlockNumber = big.NewInt(24)
	masternodes[2].CraAddress = common.HexToAddress("25c4f7736914f6dc48dcf8245e247f09c26765ff") // pass is 3

	return masternodes
}

func GetActiveMasternodes(block_number *big.Int) []*Masternode {
	return FilterNotActiveMasternodes(GetMasternodes(), block_number)
}

func GetMasternodesMap() map[common.Address]*Masternode {
	masternodes := GetMasternodes()

	masternodes_map := make(map[common.Address]*Masternode, 0)
	for _, masternode := range masternodes {
		masternodes_map[masternode.CraAddress] = masternode
	}

	return masternodes_map
}

func GetActiveMasternodesMap(block_number *big.Int) map[common.Address]*Masternode {
	masternodes := GetActiveMasternodes(block_number)

	masternodes_map := make(map[common.Address]*Masternode, 0)
	for _, masternode := range masternodes {
		masternodes_map[masternode.CraAddress] = masternode
	}

	return masternodes_map
}
