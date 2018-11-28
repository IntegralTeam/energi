package masternode

import (
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
