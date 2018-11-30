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

package masternode

import (
	"crypto/sha256"
	"errors"
	"math/big"
	"sort"
)

// Segment on rewards line. If reward point is inside this segment, this masternode is a winner
type rewardSegment struct {
	masternode *Masternode

	start *big.Int
	size  *big.Int
}

type rewardsRound struct {
	RewardsLine []rewardSegment
	Step        *big.Int // Every block, reward point moves by Step
	Length      *big.Int // Round length (line length / Step). Not equal to len(rewardSegment)!
}

// Build rewards line for current masternodes
func buildRewardsRound(masternodes []*Masternode) (*rewardsRound, error) {
	// sum collaterals
	wholeCollateral := big.NewInt(0)
	for _, masternode := range masternodes {
		wholeCollateral.Add(wholeCollateral, masternode.CollateralAmount)
	}

	// calculate roundLen
	roundLen := new(big.Int).Div(wholeCollateral, MinCollateral) // roundLen = wholeCollateral / MinCollateral
	{
		roundLenMod := new(big.Int).Mod(wholeCollateral, MinCollateral) // roundLenMod = wholeCollateral % MinCollateral
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
			segments[i].start = new(big.Int).Add(segments[i-1].start, segments[i-1].size)
		}
		segments[i].size = masternodes[i].CollateralAmount
	}

	// calculate round step
	step := new(big.Int).Div(wholeCollateral, roundLen)

	return &rewardsRound{
		segments,
		step,
		roundLen,
	}, nil
}

// Calculate a point on rewards line. The point will specify segment, segment has a masternode (winner)
func calcRewardPoint(round *rewardsRound, block_number *big.Int) *big.Int {
	roundIndex := new(big.Int).Mod(block_number, round.Length)
	roundId := new(big.Int).Sub(block_number, roundIndex) // roundId is round's first block

	roundId_hash := sha256.Sum256(roundId.Bytes())
	roundOffset := new(big.Int).SetBytes(roundId_hash[:])
	roundOffset = roundOffset.Mod(roundOffset, round.Step) // roundOffset = hash % round step

	point := new(big.Int).Mul(roundIndex, round.Step)
	point.Add(point, roundOffset) // point = index * step + offset

	return point
}

// Search for a segment which includes the point
func findPointInRound(round *rewardsRound, point *big.Int) (*Masternode, error) {
	// TODO binary search
	for _, segment := range round.RewardsLine {
		if segment.start.Cmp(point) <= 0 {
			if new(big.Int).Add(segment.start, segment.size).Cmp(point) > 0 {
				return segment.masternode, nil
			}
		}
	}

	return nil, errors.New("No masternode to reward were found")
}

// Return masternode to reward on current block
func FindWinner(masternodes []*Masternode, block_number *big.Int) (*Masternode, error) {
	activeOnly := FilterNotActiveMasternodes(masternodes, block_number)
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
