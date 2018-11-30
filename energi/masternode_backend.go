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
	"github.com/IntegralTeam/energi/energi/masternode"
	"github.com/IntegralTeam/energi/log"
	"sync"
	"time"
)

const (
	MinDurationBetweenHeartbeets = 30 * 60
	MaxDurationBetweenHeartbeets = 24 * 60 * 60
	MaxHeartbeatInFuture         = 60 * 60
	DefaultVoteExpiration        = 1000 // 1000 blocks
)

type MasternodeTrack struct {
	LastUpdated   time.Time // Local system time
	LastHeartbeat *mn_back.Heartbeat
}

type MasternodesTracker struct {
	tracks        map[common.Address]MasternodeTrack
	trackingStart time.Time // the time tracker has started

	lock sync.Mutex
}

type MasternodeBackend struct {
	Config mn_back.Config

	accountManager  *accounts.Manager
	protocolManager *ProtocolManager

	// Channel for shutting down the service
	ShutdownChan chan bool
	StoppedChan  chan bool

	heartbeatTicker  *time.Ticker
	mnWatchdogTicker *time.Ticker

	// Masternodes I want to dismiss
	MyEnemies map[common.Address][]mn_back.DismissVote

	lock sync.Mutex
}

var singleTracker *MasternodesTracker
var onceTracker sync.Once

func GetMasternodesTracker() *MasternodesTracker {
	onceTracker.Do(func() {
		singleTracker = &MasternodesTracker{
			tracks:        make(map[common.Address]MasternodeTrack),
			trackingStart: time.Now(),
		}
	})
	return singleTracker
}

var singleMnBackend *MasternodeBackend
var onceMnBackend sync.Once

func GetMasternodeBackend() *MasternodeBackend {
	return singleMnBackend
}

// Start masternode routines
// Called by: masternode
func StartMasternodeBackendAsync(config mn_back.Config, accountManager *accounts.Manager, protocolManager *ProtocolManager) (*MasternodeBackend) {
	if !config.Enabled {
		return nil
	}

	backend := MasternodeBackend{
		Config:           config,
		accountManager:   accountManager,
		protocolManager:  protocolManager,
		ShutdownChan:     make(chan bool),
		StoppedChan:      make(chan bool),
		heartbeatTicker:  time.NewTicker(time.Second * time.Duration(3*MinDurationBetweenHeartbeets)),
		mnWatchdogTicker: time.NewTicker(time.Second * time.Duration(MaxDurationBetweenHeartbeets/10)),
		MyEnemies:        make(map[common.Address][]mn_back.DismissVote),
	}

	go func(backend *MasternodeBackend) {
		log.Info("Started masternode heartbeating routine", "address", backend.Config.CraAddress.String())
		for {
			select {
			case <-backend.heartbeatTicker.C:
				backend.onHeartbeatTick()
			case <-backend.mnWatchdogTicker.C:
				backend.onMnWatchdogTick()
			case <-backend.ShutdownChan:
				log.Info("Masternode heartbeating routine stopped", "address", backend.Config.CraAddress.String())
				backend.StoppedChan <- true
				return
			}
		}
	}(&backend)

	singleMnBackend = &backend

	return &backend
}

func (backend *MasternodeBackend) Stop() {
	if singleMnBackend != nil {
		backend.ShutdownChan <- true
		<-backend.StoppedChan // wait until stopped
	}
	singleMnBackend = nil
}
