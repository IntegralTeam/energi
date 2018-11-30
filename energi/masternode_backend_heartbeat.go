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
	"github.com/IntegralTeam/energi/energi/masternode"
	"github.com/IntegralTeam/energi/log"
	"math/big"
	"time"
)

// Add heartbeat into heartbeats map
// Called by: any node
// @return true if we should relay this message
func ArrivedHeartbeat(heartbeat *mn_back.Heartbeat, block_number *big.Int) (bool, *ErrCode) {
	// Authenticate heartbeat
	craAddress, err := AuthenticateMessage(heartbeat.GetDataToSign(), &heartbeat.Auth, block_number)
	if err != nil {
		return false, err
	}

	// Get tracker
	tracker := GetMasternodesTracker()
	if (tracker == nil) {
		return true, nil
	}
	tracker.lock.Lock()
	defer tracker.lock.Unlock()

	// Heartbeat too-far-in-future check
	now := time.Now()
	{
		now_tmp := now
		if now_tmp.Add(time.Second * time.Duration(MaxHeartbeatInFuture)).Before(heartbeat.Time()) {
			log.Debug("Received a too-far-in-future heartbeat", "address", craAddress)
			return false, &ErrCode{ErrMsgTooFarInFuture}
		}
	}

	// Heartbeat too-early check
	track, ok := tracker.tracks[craAddress]
	if ok {
		// If new heartbeat came too early, just drop it and return false
		prevHeartbeatTime := track.LastHeartbeat.Time()
		if prevHeartbeatTime.Add(time.Second * time.Duration(MinDurationBetweenHeartbeets)).After(heartbeat.Time()) {
			log.Debug("Received a too-early heartbeat", "address", craAddress)
			return false, nil
		}
	}
	// Save heartbeat
	tracker.tracks[craAddress] = MasternodeTrack{
		LastUpdated:   now,
		LastHeartbeat: heartbeat,
	}
	return true, nil
}

// Send heartbeat (to prove this MN is alive)
func (backend *MasternodeBackend) onHeartbeatTick() {
	backend.lock.Lock()
	defer backend.lock.Unlock()

	if !backend.amIActiveMasternode() {
		return
	}

	heartbeat := mn_back.Heartbeat{}
	heartbeat.Timestamp = uint64(time.Now().Unix())

	// Sign the heartbeat
	err := backend.signMessage(heartbeat.GetDataToSign(), &heartbeat.Auth)
	if err != nil {
		log.Error("Unable to sign the heartbeat", "err", err, "address", backend.Config.CraAddress.String())
		return
	}

	// Broadcast the heartbeat
	log.Info("Broadcasting masternode heartbeat", "address", backend.Config.CraAddress.String())
	backend.protocolManager.BroadcastHeartbeat(&heartbeat)
}
