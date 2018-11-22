package masternode_reward

import (
	"crypto/sha256"
	"errors"
	"github.com/IntegralTeam/energi/common"
	"github.com/IntegralTeam/energi/p2p/discv5"
	"github.com/IntegralTeam/energi/params"
	"math/big"
	"sort"
)

// Represents Masternode. This state is stored inside masternodes smart contract.
type Masternode struct {
	Alias string // human-readable name

	// net addresses
	NodeAddressIpV4 *discv5.NodeAddress
	NodeAddressIpV6 *discv5.NodeAddress // Optional network address

	CollateralAmount *big.Int
	CraAddress common.Address // CRA (Collateral/Reward/Authentication) address. The address from which the collateral was sent
	AnnouncementBlockNumber *big.Int // The block in which the tx-Announce was included
	ActivationBlockNumber *big.Int // Formula: <Announcement block number> + max(round_up(<whole collateral> / <MinCollateral>), 100)
}

// Minimum masternode collateral
var MinCollateral = big.NewInt(0).Mul(big.NewInt(10000), params.Energi_bn) // 10000 NRG

// Segment on rewards line. If reward point is inside this segment, this masternode is a winner
type rewardSegment struct {
	masternode *Masternode

	start *big.Int
	size *big.Int
}

type RewardsRound struct {
	RewardsLine []rewardSegment
	Step *big.Int // Every block, reward point moves by Step
	Length *big.Int // Round length (line length / Step). Not equal to len(rewardSegment)!
}

// Return only activated masternodes
func filterNotActiveMasternodes(masternodes []*Masternode) []*Masternode {
	masternodesFiltered := make([]*Masternode, 0, len(masternodes))

	for _, masternode := range masternodes {
		if masternode.AnnouncementBlockNumber.Cmp(masternode.ActivationBlockNumber) >= 0 {
			masternodesFiltered = append(masternodesFiltered, masternode)
		}
	}
	return masternodesFiltered
}

// Build rewards line for current masternodes
func buildRewardsRound(masternodes []*Masternode) (*RewardsRound, error) {
	// sum collaterals
	wholeCollateral := big.NewInt(0)
	for _, masternode := range masternodes {
		wholeCollateral.Add(wholeCollateral, masternode.CollateralAmount)
	}

	// calculate roundLen
	roundLen := big.NewInt(0).Div(wholeCollateral, MinCollateral) // roundLen = wholeCollateral / MinCollateral
	{
		roundLenMod := big.NewInt(0).Mod(wholeCollateral, MinCollateral) // roundLenMod = wholeCollateral % MinCollateral
		if roundLenMod.Cmp(big.NewInt(0)) != 0 { // roundLen = round up wholeCollateral / MinCollateral
			roundLen.Add(roundLen, big.NewInt(1))
		}
	}

	if !roundLen.IsUint64() {
		return nil, errors.New("Too long masternodes rewards round (longer than 2^64 blocks)")
	}

	// sort masternodes by their age
	sort.Slice(masternodes, func(i, j int) bool {
		return masternodes[i].AnnouncementBlockNumber.Cmp(masternodes[j].AnnouncementBlockNumber) < 0
	})

	// calculate round segments
	segments := make([]rewardSegment, len(masternodes), len(masternodes))
	for i, _ := range masternodes {
		segments[i].masternode = masternodes[i]
		if i == 0 {
			segments[i].start = big.NewInt(0)
		} else {
			segments[i].start = big.NewInt(0).Add(segments[i - 1].start, segments[i - 1].size)
		}
		segments[i].size = masternodes[i].CollateralAmount
	}

	// calculate round step
	step := big.NewInt(0).Div(wholeCollateral, roundLen)

	return &RewardsRound{
		segments,
		step,
		roundLen,
	}, nil
}

// Calculate a point on rewards line. The point will specify segment, segment has a masternode (winner)
func calcRewardPoint(round *RewardsRound, block_number *big.Int) *big.Int {
	roundIndex := big.NewInt(0).Mod(block_number, round.Length)
	roundId := big.NewInt(0).Sub(block_number, roundIndex) // roundId is round's first block

	roundId_hash := sha256.Sum256(roundId.Bytes())
	roundOffset := big.NewInt(0).SetBytes(roundId_hash[:])
	roundOffset = roundOffset.Mod(roundOffset, round.Step) // roundOffset = hash % round step

	point := big.NewInt(0).Mul(roundIndex, round.Step)
	point.Add(point, roundOffset) // point = index * step + offset

	return point
}

// Search for a segment which includes the point
func findPointInRound(round *RewardsRound, point *big.Int) (*Masternode, error) {
	// TODO binary search
	for _, segment := range round.RewardsLine {
		if segment.start.Cmp(point) <= 0 {
			if big.NewInt(0).Add(segment.start, segment.size).Cmp(point) > 0 {
				return segment.masternode, nil
			}
		}
	}

	return nil, errors.New("No masternode to reward were found")
}

// Return masternode to reward on current block
func FindWinner(masternodes []*Masternode, block_number *big.Int) (*Masternode, error) {
	activeOnly := filterNotActiveMasternodes(masternodes, block_number)
	if len(activeOnly) == 0 {
		return nil, errors.New("No masternode to reward were found")
	}

	round, err := buildRewardsRound(activeOnly)
	if err != nil {
		return nil, err
	}

	rewardPoint := calcRewardPoint(round, block_number)
	winner, err := findPointInRound(round, rewardPoint)
	if err != nil {
		return nil, err
	}
	return winner, nil
}
