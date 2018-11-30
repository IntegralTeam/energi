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
	"github.com/IntegralTeam/energi/common"
	"github.com/IntegralTeam/energi/consensus/masternode"
	"github.com/IntegralTeam/energi/energi/masternode"
	"github.com/IntegralTeam/energi/log"
	"math/big"
	"time"
)

// Check for dead masternodes. Vote/re-vote against dead ones
func (backend *MasternodeBackend) onMnWatchdogTick() {
	backend.lock.Lock()
	defer backend.lock.Unlock()

	if !backend.amIActiveMasternode() {
		return
	}

	block_number := backend.protocolManager.blockchain.CurrentBlock().Header().Number
	masternodes := mn.GetActiveMasternodes(block_number)

	tracker := GetMasternodesTracker()
	tracker.lock.Lock()
	defer tracker.lock.Unlock()

	for _, mn := range masternodes {
		if mn.CraAddress == backend.Config.CraAddress {
			continue // don't vote against myself
		}

		lastHeartbeatTime := time.Time{}

		track, ok := tracker.tracks[mn.CraAddress]
		if ok {
			lastHeartbeatTime = track.LastHeartbeat.Time()
		} else {
			lastHeartbeatTime = tracker.trackingStart
		}

		now := time.Now()
		if now.After(lastHeartbeatTime.Add(time.Second * time.Duration(MaxDurationBetweenHeartbeets))) {
			// This masternode didn't have heartbeats for long time. It's my enemy!
			vote := mn_back.DismissVote{}

			vote.ExpirationBlockNumber = new(big.Int).Add(block_number, big.NewInt(DefaultVoteExpiration))
			vote.CraAddressToDismiss = mn.CraAddress
			vote.Reason.Code = mn_back.DissmissVote_NoHeartbeats
			vote.Reason.Description = "Automatically generated vote against a dead masternode"
			vote.Timestamp = uint64(now.Unix())
			err := backend.signMessage(vote.GetDataToSign(), &vote.Auth)
			if err != nil {
				log.Error("Unable to sign the vote against a dead masternode", "err", err, "address", backend.Config.CraAddress.String())
				return
			}

			backend.AddMyEnemy(&vote)
		} else {
			log.Debug("Masternode has enough heartbeats", "address", mn.CraAddress.String())
			backend.ForgiveMyEnemy(mn.CraAddress, mn_back.DissmissVote_NoHeartbeats)
		}
	}
}

// Arrived foreign vote against some node. I need to authenticate it.
// If it's my enemy, check current number of votes.
// Called by: any node
// @return true if we should relay this message
func ArrivedDismmisingVote(vote *mn_back.DismissVote, block_number *big.Int) (bool, *ErrCode) {
	// Authenticate vote
	_, err := AuthenticateMessage(vote.GetDataToSign(), &vote.Auth, block_number)
	if err != nil {
		return false, err
	}

	// Get MN backend
	backend := GetMasternodeBackend()
	if (backend == nil) { // I'm not a masternode
		return true, nil
	}
	backend.lock.Lock()
	defer backend.lock.Unlock()

	// Process vote
	backend.clearOutdatedDissmissVotes()
	inserted, _ := backend.insertDismissVote(vote, false)
	backend.clearOutdatedDissmissVotes()
	backend.executeFulfilledDismissVotings()

	return inserted, nil
}

// Add to the watchlist of nodes which I want to dismiss
// backend.lock is supposed to be locked
// Called by: masternode
func (backend *MasternodeBackend) AddMyEnemy(vote *mn_back.DismissVote) {
	backend.clearOutdatedDissmissVotes()
	inserted, _ := backend.insertDismissVote(vote, true)
	backend.clearOutdatedDissmissVotes()
	backend.executeFulfilledDismissVotings()

	if inserted {
		// Broadcast the vote
		log.Info("Broadcasting masternode dismiss vote", "address", backend.Config.CraAddress.String(), "against", vote.CraAddressToDismiss.String())
		backend.protocolManager.BroadcastDismissVote(vote)
	}
}

// Erase from the watchlist of nodes which I want to dismiss
// backend.lock is supposed to be locked
// Called by: masternode
func (backend *MasternodeBackend) ForgiveMyEnemy(craAddressToForgive common.Address, reasonCode mn_back.DismissingReasonCode) {
	votes, ok := backend.MyEnemies[craAddressToForgive]
	if !ok {
		return
	}

	block_number := backend.protocolManager.blockchain.CurrentBlock().Header().Number

	// Erase my vote with this code (there's no more than 1 vote on pair address/reasonCode)
	for i, existingVote := range votes {
		existingSenderCraAddress, _ := AuthenticateMessage(existingVote.GetDataToSign(), &existingVote.Auth, block_number)
		if existingSenderCraAddress == backend.Config.CraAddress && existingVote.Reason.Code == reasonCode {
			// Erase vote
			votes = append(votes[:i], votes[i+1:]...)
			break
		}
	}

	// Do I still have votes with another code? If yes, then it's still an enemy
	stillEnemy := false
	for _, existingVote := range votes {
		existingSenderCraAddress, _ := AuthenticateMessage(existingVote.GetDataToSign(), &existingVote.Auth, block_number)
		if existingSenderCraAddress == backend.Config.CraAddress {
			stillEnemy = true
			break
		}
	}
	if !stillEnemy {
		delete(backend.MyEnemies, craAddressToForgive)
	}
}

// @param myVote is false, then don't insert if it's not in my watchlist
// backend.lock is supposed to be locked
func (backend *MasternodeBackend) insertDismissVote(vote *mn_back.DismissVote, myVote bool) (inserted bool, existed bool) {
	votes, ok := backend.MyEnemies[vote.CraAddressToDismiss]
	if !ok && !myVote {
		return false, false
	}

	block_number := backend.protocolManager.blockchain.CurrentBlock().Header().Number

	// Try to update existing vote
	existed = false
	for i, existingVote := range votes {
		senderCraAddress, _ := AuthenticateMessage(vote.GetDataToSign(), &vote.Auth, block_number)
		existingSenderCraAddress, _ := AuthenticateMessage(existingVote.GetDataToSign(), &existingVote.Auth, block_number)
		if existingSenderCraAddress == senderCraAddress && existingVote.Reason.Code == vote.Reason.Code {
			// Update vote
			votes[i] = *vote
			existed = true
		}
	}

	if !existed { // insert it otherwise
		votes, ok := backend.MyEnemies[vote.CraAddressToDismiss]
		if !ok {
			backend.MyEnemies[vote.CraAddressToDismiss] = make([]mn_back.DismissVote, 0, 1)
			votes = backend.MyEnemies[vote.CraAddressToDismiss]
		}
		votes = append(votes, *vote)
		backend.MyEnemies[vote.CraAddressToDismiss] = votes
	}

	return true, existed
}

// @param myVote is false, then don't insert if it's not in my watchlist
// backend.lock is supposed to be locked
func (backend *MasternodeBackend) executeFulfilledDismissVotings() {
	readyToBeDissmissed := backend.findFulfilledDismissVotings()

	for _, craAddressToDismiss := range readyToBeDissmissed {
		log.Info("Dismissing", "address", backend.Config.CraAddress.String(), "against", craAddressToDismiss.String())
	}
}

func GetMinDismissingQuorum(numberOfMasternodes uint64) uint64 {
	if (numberOfMasternodes <= 8) {
		return 7
	}
	return (numberOfMasternodes * 9) / 10 // 90%, rounding down
}

// backend.lock is supposed to be locked
// Called by: masternode
func (backend *MasternodeBackend) findFulfilledDismissVotings() []common.Address {
	block_number := backend.protocolManager.blockchain.CurrentBlock().Header().Number
	masternodes_number := uint64(len(mn.GetActiveMasternodes(block_number)))

	res := make([]common.Address, 0, 4)
	for craAddressToDismiss, votes := range backend.MyEnemies {
		if uint64(len(votes)) >= GetMinDismissingQuorum(masternodes_number) {
			res = append(res, craAddressToDismiss)
		}
	}
	return res
}

// backend.lock is supposed to be locked
// Called by: masternode
func (backend *MasternodeBackend) clearOutdatedDissmissVotes() {
	block_number := backend.protocolManager.blockchain.CurrentBlock().Header().Number

	for _, votes := range backend.MyEnemies {
	Repeat:
		for i, existingVote := range votes {
			_, err := AuthenticateMessage(existingVote.GetDataToSign(), &existingVote.Auth, block_number)
			if err != nil { // vote author isn't authenticated anymore
				votes = append(votes[:i], votes[i+1:]...)
				goto Repeat // we erase while iterating
			}
			if existingVote.ExpirationBlockNumber.Cmp(block_number) >= 0 {
				votes = append(votes[:i], votes[i+1:]...) // vote expired
				goto Repeat                               // we erase while iterating
			}
		}
	}
}
