package masternode_reward

import (
	"fmt"
	"github.com/IntegralTeam/energi/common"
	"github.com/IntegralTeam/energi/params"
	"github.com/magiconair/properties/assert"
	"math/big"
	"testing"
)

func getTestMasternode_1() []*Masternode {
	masternodes := make([]*Masternode, 1, 1)

	for i, _ := range masternodes {
		masternodes[i] = &Masternode{
			Alias : fmt.Sprintf("MN%d", i),
			NodeAddressIpV4 : nil,
			NodeAddressIpV6 : nil,
			CollateralAmount : new(big.Int),
			CraAddress : common.Address{},
			AnnouncementBlockNumber : new(big.Int),
			ActivationBlockNumber : new(big.Int),
		}
	}
	masternodes[0].CollateralAmount = big.NewInt(0).Mul(big.NewInt(10000), params.Energi_bn)
	masternodes[0].AnnouncementBlockNumber = big.NewInt(0)
	masternodes[0].ActivationBlockNumber = big.NewInt(10)

	return masternodes
}

func getTestMasternodes_3_normal() []*Masternode {
	masternodes := make([]*Masternode, 3, 3)

	for i, _ := range masternodes {
		masternodes[i] = &Masternode{
			Alias : fmt.Sprintf("MN%d", i),
			NodeAddressIpV4 : nil,
			NodeAddressIpV6 : nil,
			CollateralAmount : new(big.Int),
			CraAddress : common.Address{},
			AnnouncementBlockNumber : new(big.Int),
			ActivationBlockNumber : new(big.Int),
		}
	}
	masternodes[0].CollateralAmount = big.NewInt(0).Mul(big.NewInt(10001), params.Energi_bn)
	masternodes[0].AnnouncementBlockNumber = big.NewInt(0)
	masternodes[0].ActivationBlockNumber = big.NewInt(4)

	masternodes[1].CollateralAmount = big.NewInt(0).Mul(big.NewInt(20000), params.Energi_bn)
	masternodes[1].AnnouncementBlockNumber = big.NewInt(10)
	masternodes[1].ActivationBlockNumber = big.NewInt(14)

	masternodes[2].CollateralAmount = big.NewInt(0).Mul(big.NewInt(30000), params.Energi_bn)
	masternodes[2].AnnouncementBlockNumber = big.NewInt(20)
	masternodes[2].ActivationBlockNumber = big.NewInt(24)

	return masternodes
}

func getTestMasternodes_3_noReminder() []*Masternode {
	masternodes := make([]*Masternode, 3, 3)

	masternodes[0].CollateralAmount = big.NewInt(0).Mul(big.NewInt(10000), params.Energi_bn)

	return masternodes
}

func getTestMasternodes_3_reversed() []*Masternode {
	masternodes := make([]*Masternode, 3, 3)

	masternodes[0].AnnouncementBlockNumber = big.NewInt(0)
	masternodes[0].ActivationBlockNumber = big.NewInt(4)

	masternodes[1].AnnouncementBlockNumber = big.NewInt(1)
	masternodes[1].ActivationBlockNumber = big.NewInt(3)

	masternodes[2].AnnouncementBlockNumber = big.NewInt(2)
	masternodes[2].ActivationBlockNumber = big.NewInt(2)

	return masternodes
}

func getTestMasternodes_same(num int) []*Masternode {
	masternodes := make([]*Masternode, num, num)

	for i, _ := range masternodes {
		masternodes[i] = &Masternode{
			Alias : fmt.Sprintf("MN%d", i),
			NodeAddressIpV4 : nil,
			NodeAddressIpV6 : nil,
			CollateralAmount : big.NewInt(0).Mul(big.NewInt(10001), params.Energi_bn),
			CraAddress : common.Address{},
			AnnouncementBlockNumber : big.NewInt(0),
			ActivationBlockNumber : big.NewInt(4),
		}
	}

	return masternodes
}

func getTestMasternodes_same_noReminder(num int) []*Masternode {
	masternodes := make([]*Masternode, num, num)

	for i, _ := range masternodes {
		masternodes[i] = &Masternode{
			Alias : fmt.Sprintf("MN%d", i),
			NodeAddressIpV4 : nil,
			NodeAddressIpV6 : nil,
			CollateralAmount : big.NewInt(0).Mul(big.NewInt(10000), params.Energi_bn),
			CraAddress : common.Address{},
			AnnouncementBlockNumber : big.NewInt(0),
			ActivationBlockNumber : big.NewInt(4),
		}
	}

	return masternodes
}

func Test_filterNotActiveMasternodes_3_normal(t *testing.T) {
	masternodes := getTestMasternodes_3_normal()
	masternodes[2].ActivationBlockNumber.Mul(big.NewInt(1e+16), big.NewInt(1e+16))

	masternodes_filtered0 := filterNotActiveMasternodes(masternodes, big.NewInt(0))
	masternodes_filtered1 := filterNotActiveMasternodes(masternodes, big.NewInt(1))
	masternodes_filtered2 := filterNotActiveMasternodes(masternodes, big.NewInt(2))
	masternodes_filtered3 := filterNotActiveMasternodes(masternodes, big.NewInt(3))
	masternodes_filtered4 := filterNotActiveMasternodes(masternodes, big.NewInt(4))
	masternodes_filtered5 := filterNotActiveMasternodes(masternodes, big.NewInt(5))

	masternodes_filtered13 := filterNotActiveMasternodes(masternodes, big.NewInt(13))
	masternodes_filtered14 := filterNotActiveMasternodes(masternodes, big.NewInt(14))
	masternodes_filtered15 := filterNotActiveMasternodes(masternodes, big.NewInt(15))

	masternodes_filtered1e32sub := filterNotActiveMasternodes(masternodes, new(big.Int).Sub(masternodes[2].ActivationBlockNumber, big.NewInt(1)))
	masternodes_filtered1e32 := filterNotActiveMasternodes(masternodes, masternodes[2].ActivationBlockNumber)
	masternodes_filtered1e32sum := filterNotActiveMasternodes(masternodes, new(big.Int).Add(masternodes[2].ActivationBlockNumber, big.NewInt(1)))

	// Ensure right number of entries were filtered
	assert.Equal(t, len(masternodes_filtered0), 0)
	assert.Equal(t, len(masternodes_filtered1), 0)
	assert.Equal(t, len(masternodes_filtered2), 0)
	assert.Equal(t, len(masternodes_filtered3), 0)
	assert.Equal(t, len(masternodes_filtered4), 1)
	assert.Equal(t, len(masternodes_filtered5), 1)

	assert.Equal(t, len(masternodes_filtered13), 1)
	assert.Equal(t, len(masternodes_filtered14), 2)
	assert.Equal(t, len(masternodes_filtered15), 2)

	assert.Equal(t, len(masternodes_filtered1e32sub), 2)
	assert.Equal(t, len(masternodes_filtered1e32), 3)
	assert.Equal(t, len(masternodes_filtered1e32sum), 3)

	// Ensure right entries were returned
	assert.Equal(t, masternodes_filtered4[0].Alias, "MN0")
	assert.Equal(t, masternodes_filtered4[0], masternodes_filtered5[0])
	assert.Equal(t, masternodes_filtered4[0], masternodes_filtered13[0])
	assert.Equal(t, masternodes_filtered4[0], masternodes_filtered14[0])
	assert.Equal(t, masternodes_filtered4[0], masternodes_filtered15[0])
	assert.Equal(t, masternodes_filtered4[0], masternodes_filtered1e32sub[0])
	assert.Equal(t, masternodes_filtered4[0], masternodes_filtered1e32[0])
	assert.Equal(t, masternodes_filtered4[0], masternodes_filtered1e32sum[0])

	assert.Equal(t, masternodes_filtered14[1].Alias, "MN1")
	assert.Equal(t, masternodes_filtered14[1], masternodes_filtered15[1])
	assert.Equal(t, masternodes_filtered14[1], masternodes_filtered1e32sub[1])
	assert.Equal(t, masternodes_filtered14[1], masternodes_filtered1e32[1])
	assert.Equal(t, masternodes_filtered14[1], masternodes_filtered1e32sum[1])

	assert.Equal(t, masternodes_filtered1e32[2].Alias, "MN2")
	assert.Equal(t, masternodes_filtered1e32[2], masternodes_filtered1e32sum[2])
}