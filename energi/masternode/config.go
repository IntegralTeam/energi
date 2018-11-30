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

import "github.com/IntegralTeam/energi/common"

//go:generate gencodec -type Config -formats toml -out gen_config.go

type Config struct {
	Enabled    bool   // Enabled means it's a masternode
	Passphrase string // passpharase for a wallet which holds CraAddress
	CraAddress common.Address
}
