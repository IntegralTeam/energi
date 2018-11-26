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
	masternodes[0].CollateralAmount = new(big.Int).Mul(big.NewInt(10000), params.Energi_bn)
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
	masternodes[0].CollateralAmount = new(big.Int).Mul(big.NewInt(10001), params.Energi_bn)
	masternodes[0].AnnouncementBlockNumber = big.NewInt(0)
	masternodes[0].ActivationBlockNumber = big.NewInt(4)

	masternodes[1].CollateralAmount = new(big.Int).Mul(big.NewInt(10000), params.Energi_bn)
	masternodes[1].AnnouncementBlockNumber = big.NewInt(10)
	masternodes[1].ActivationBlockNumber = big.NewInt(14)

	masternodes[2].CollateralAmount = new(big.Int).Mul(big.NewInt(10000), params.Energi_bn)
	masternodes[2].AnnouncementBlockNumber = big.NewInt(20)
	masternodes[2].ActivationBlockNumber = big.NewInt(24)

	return masternodes
}

func getTestMasternodes_3_noReminder() []*Masternode {
	masternodes := getTestMasternodes_3_normal()

	masternodes[0].CollateralAmount = new(big.Int).Mul(big.NewInt(10000), params.Energi_bn)

	return masternodes
}

func getTestMasternodes_3_reversed() []*Masternode {
	masternodes := getTestMasternodes_3_normal()

	masternodes[0].AnnouncementBlockNumber = big.NewInt(0)
	masternodes[0].ActivationBlockNumber = big.NewInt(4)

	masternodes[1].AnnouncementBlockNumber = big.NewInt(1)
	masternodes[1].ActivationBlockNumber = big.NewInt(3)

	masternodes[2].AnnouncementBlockNumber = big.NewInt(2)
	masternodes[2].ActivationBlockNumber = big.NewInt(2)

	return masternodes
}

func getTestMasternodes_increasingCollateral(num int) []*Masternode {
	masternodes := make([]*Masternode, num, num)

	for i, _ := range masternodes {
		masternodes[i] = &Masternode{
			Alias : fmt.Sprintf("MN%d", i),
			NodeAddressIpV4 : nil,
			NodeAddressIpV6 : nil,
			CollateralAmount : new(big.Int).Mul(big.NewInt(int64(i * 10000)), params.Energi_bn),
			CraAddress : common.Address{},
			AnnouncementBlockNumber : big.NewInt(int64(i)),
			ActivationBlockNumber : big.NewInt(int64(i + 1)),
		}
	}

	return masternodes
}

func getTestMasternodes_noReminder(num int) []*Masternode {
	masternodes := make([]*Masternode, num, num)

	for i, _ := range masternodes {
		masternodes[i] = &Masternode{
			Alias : fmt.Sprintf("MN%d", i),
			NodeAddressIpV4 : nil,
			NodeAddressIpV6 : nil,
			CollateralAmount : new(big.Int).Mul(big.NewInt(10000), params.Energi_bn),
			CraAddress : common.Address{},
			AnnouncementBlockNumber : big.NewInt(int64(i)),
			ActivationBlockNumber : big.NewInt(int64(i + 1)),
		}
	}

	return masternodes
}

func test_no_winners(t *testing.T, masternodes []*Masternode, until int) {
	// test that there's no winner until masternode activation
	for block_i := 0; block_i < until; block_i++ {
		_, err := FindWinner(masternodes, big.NewInt(int64(block_i)))
		assert.Equal(t, err.Error(), "No masternode to reward were found")
	}
}

func test_winner_is(t *testing.T, masternodes []*Masternode, block_number int, winner_want *Masternode) {
	block_i := big.NewInt(int64(block_number))

	winner, err := FindWinner(masternodes, block_i)
	assert.Equal(t, err, nil)
	assert.Equal(t, winner, winner_want)
}

func Test_FindWinner_1(t *testing.T) {
	masternodes := getTestMasternode_1()
	test_no_winners(t, masternodes, 10)
	// test that the masternode is always a winner because it's the only masternode
	for block_i := 10; block_i < 1000; block_i++ {
		winner, err := FindWinner(masternodes, big.NewInt(int64(block_i)))
		assert.Equal(t, err, nil)
		assert.Equal(t, winner, masternodes[0])
	}
}

func test_fifo_rewards(t *testing.T, start_from int, masternodes []*Masternode) {
	for block_i := start_from; block_i < start_from + 1000; block_i++ {
		winner, err := FindWinner(masternodes, big.NewInt(int64(block_i)))
		assert.Equal(t, err, nil)
		assert.Equal(t, winner, masternodes[block_i % len(masternodes)])
	}
}

// Test average distribution of rewards
func Test_buildRewardsRound_distribution(t *testing.T) {
	masternodes := getTestMasternodes_increasingCollateral(50)

	masternodeHits := make(map[int]int) // masternode -> number of occurrences
	for block_i := 50; block_i < 10000; block_i++ {
		winner, _ := FindWinner(masternodes, big.NewInt(int64(block_i)))
		collateral_factor := new(big.Int).Div(winner.CollateralAmount, MinCollateral).Uint64()

		_, ok := masternodeHits[int(collateral_factor)]
		if !ok {
			masternodeHits[int(collateral_factor)] = 0
		}
		masternodeHits[int(collateral_factor)] += 1
	}

	for collateral_factor, hits := range masternodeHits {
		fmt.Printf("Test_buildRewardsRound_distribution: %d \n", hits / collateral_factor)
		assert.Equal(t, (hits / collateral_factor) > 8 - 2, true)
		assert.Equal(t, (hits / collateral_factor) < 8 + 2, true)
	}
}

// Test that rewards work like FIFO when all the masternodes have the same collateral
func Test_buildRewardsRound_no_reminder(t *testing.T) {
	masternodes := getTestMasternodes_noReminder(2)
	test_fifo_rewards(t, 2, masternodes)

	masternodes = getTestMasternodes_noReminder(5)
	test_fifo_rewards(t, 5, masternodes)

	masternodes = getTestMasternodes_noReminder(10)
	test_fifo_rewards(t, 10, masternodes)

	masternodes = getTestMasternodes_noReminder(100)
	test_fifo_rewards(t, 100, masternodes)

	masternodes = getTestMasternodes_noReminder(1000)
	test_fifo_rewards(t, 1000, masternodes)

	masternodes = getTestMasternodes_noReminder(10000)
	test_fifo_rewards(t, 10000, masternodes)
}

// Test that masternodes are sorted by their age
func Test_buildRewardsRound_sorting(t *testing.T) {
	masternodes := getTestMasternodes_3_reversed()
	masternodes[0], masternodes[1] = masternodes[1], masternodes[0]

	round, _ := buildRewardsRound(masternodes)

	assert.Equal(t, round.RewardsLine[0].masternode.Alias, "MN0")
	assert.Equal(t, round.RewardsLine[1].masternode.Alias, "MN1")
	assert.Equal(t, round.RewardsLine[2].masternode.Alias, "MN2")
}

func Test_FindWinner_3_normal(t *testing.T) {
	masternodes := getTestMasternodes_3_normal()
	test_no_winners(t, masternodes, 4)

	// Test 4-13 blocks. First masternode is always a winner
	for i := 4; i < 14; i++ {
		activeOnly := filterNotActiveMasternodes(masternodes, big.NewInt(int64(i)))

		round, err := buildRewardsRound(activeOnly)
		assert.Equal(t, err, nil)
		assert.Equal(t, round.Length.Uint64(), uint64(2))
		assert.Equal(t, round.Step.Cmp(new(big.Int).Div(masternodes[0].CollateralAmount, big.NewInt(2))), 0)

		assert.Equal(t, len(round.RewardsLine), 1)
		assert.Equal(t, round.RewardsLine[0].start.Uint64(), uint64(0))
		assert.Equal(t, round.RewardsLine[0].size.Cmp(masternodes[0].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[0].masternode, masternodes[0])


		winner, err := FindWinner(masternodes, big.NewInt(int64(i)))
		assert.Equal(t, err, nil)
		assert.Equal(t, winner, masternodes[0])
	}

	// Test 14 block. 2 activated masternodes
	{
		block_i := big.NewInt(14)

		activeOnly := filterNotActiveMasternodes(masternodes, block_i)

		round, err := buildRewardsRound(activeOnly)
		assert.Equal(t, err, nil)
		assert.Equal(t, round.Length.Uint64(), uint64(3))

		assert.Equal(t, len(round.RewardsLine), 2)
		assert.Equal(t, round.RewardsLine[0].start.Uint64(), uint64(0))
		assert.Equal(t, round.RewardsLine[0].size.Cmp(masternodes[0].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[0].masternode, masternodes[0])

		assert.Equal(t, round.RewardsLine[1].start.Cmp(masternodes[0].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[1].size.Cmp(masternodes[1].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[1].masternode, masternodes[1])

		winner, err := FindWinner(masternodes, block_i)
		assert.Equal(t, err, nil)
		assert.Equal(t, winner, masternodes[1]) // 14 % 3 = 2, last step, MN1 is always is a winner
	}

	// Test 15-24 blocks
	test_winner_is(t, masternodes, 15, masternodes[0]) // 15 % 3 = 0, first step, MN0 is always is a winner
	test_winner_is(t, masternodes, 16, masternodes[1]) // 1 step
	test_winner_is(t, masternodes, 17, masternodes[1]) // 2 step
	test_winner_is(t, masternodes, 18, masternodes[0]) // 0 step
	test_winner_is(t, masternodes, 19, masternodes[1]) // 1 step
	test_winner_is(t, masternodes, 20, masternodes[1]) // 2 step
	test_winner_is(t, masternodes, 21, masternodes[0]) // 0 step
	test_winner_is(t, masternodes, 22, masternodes[0]) // 1 step
	test_winner_is(t, masternodes, 23, masternodes[1]) // 2 step
	test_winner_is(t, masternodes, 24, masternodes[0]) // 0 step

	// Test 25 block. 3 activated masternodes
	{
		block_i := big.NewInt(25)

		activeOnly := filterNotActiveMasternodes(masternodes, block_i)

		round, err := buildRewardsRound(activeOnly)
		assert.Equal(t, err, nil)
		assert.Equal(t, round.Length.Uint64(), uint64(4))

		assert.Equal(t, len(round.RewardsLine), 3)
		assert.Equal(t, round.RewardsLine[0].start.Uint64(), uint64(0))
		assert.Equal(t, round.RewardsLine[0].size.Cmp(masternodes[0].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[0].masternode, masternodes[0])

		assert.Equal(t, round.RewardsLine[1].start.Cmp(masternodes[0].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[1].size.Cmp(masternodes[1].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[1].masternode, masternodes[1])

		assert.Equal(t, round.RewardsLine[2].start.Cmp(new(big.Int).Add(round.RewardsLine[1].start, round.RewardsLine[1].size)), 0)
		assert.Equal(t, round.RewardsLine[2].size.Cmp(masternodes[2].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[2].masternode, masternodes[2])

		winner, err := FindWinner(masternodes, block_i)
		assert.Equal(t, err, nil)
		assert.Equal(t, winner, masternodes[0]) // 25 % 4 = 1, second step
	}

	// Test 26-39 blocks
	test_winner_is(t, masternodes, 26, masternodes[1]) // 2 step
	test_winner_is(t, masternodes, 27, masternodes[2]) // 3 step
	test_winner_is(t, masternodes, 28, masternodes[0]) // 0 step
	test_winner_is(t, masternodes, 29, masternodes[1]) // 1 step
	test_winner_is(t, masternodes, 30, masternodes[1]) // 2 step
	test_winner_is(t, masternodes, 31, masternodes[2]) // 3 step
	test_winner_is(t, masternodes, 32, masternodes[0]) // 0 step
	test_winner_is(t, masternodes, 33, masternodes[0]) // 1 step
	test_winner_is(t, masternodes, 34, masternodes[1]) // 2 step
	test_winner_is(t, masternodes, 35, masternodes[2]) // 3 step
	test_winner_is(t, masternodes, 36, masternodes[0]) // 0 step
	test_winner_is(t, masternodes, 37, masternodes[1]) // 1 step
	test_winner_is(t, masternodes, 38, masternodes[2]) // 2 step
	test_winner_is(t, masternodes, 39, masternodes[2]) // 3 step
}

func Test_FindWinner_3_noReminder(t *testing.T) {
	masternodes := getTestMasternodes_3_noReminder()
	test_no_winners(t, masternodes, 4)

	// Test 14 block. 2 activated masternodes
	{
		block_i := big.NewInt(14)

		activeOnly := filterNotActiveMasternodes(masternodes, block_i)

		round, err := buildRewardsRound(activeOnly)
		assert.Equal(t, err, nil)
		assert.Equal(t, round.Length.Uint64(), uint64(2))
		assert.Equal(t, round.Step.Cmp(new(big.Int).Mul(big.NewInt(10000), params.Energi_bn)), 0)

		assert.Equal(t, len(round.RewardsLine), 2)
		assert.Equal(t, round.RewardsLine[0].start.Uint64(), uint64(0))
		assert.Equal(t, round.RewardsLine[0].size.Cmp(masternodes[0].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[0].masternode, masternodes[0])

		assert.Equal(t, round.RewardsLine[1].start.Cmp(masternodes[0].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[1].size.Cmp(masternodes[1].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[1].masternode, masternodes[1])

		winner, err := FindWinner(masternodes, block_i)
		assert.Equal(t, err, nil)
		assert.Equal(t, winner, masternodes[0]) // 14 % 2 = 0, first step, MN0 is always is a winner
	}

	// Test 15-24 blocks
	test_winner_is(t, masternodes, 15, masternodes[1]) // 1 step
	test_winner_is(t, masternodes, 16, masternodes[0]) // 0 step
	test_winner_is(t, masternodes, 17, masternodes[1]) // 1 step
	test_winner_is(t, masternodes, 18, masternodes[0]) // 0 step
	test_winner_is(t, masternodes, 19, masternodes[1]) // 1 step
	test_winner_is(t, masternodes, 20, masternodes[0]) // 0 step
	test_winner_is(t, masternodes, 21, masternodes[1]) // 1 step
	test_winner_is(t, masternodes, 22, masternodes[0]) // 0 step
	test_winner_is(t, masternodes, 23, masternodes[1]) // 1 step
	test_winner_is(t, masternodes, 24, masternodes[0]) // 0 step

	// Test 25 block. 3 activated masternodes
	{
		block_i := big.NewInt(25)

		activeOnly := filterNotActiveMasternodes(masternodes, block_i)

		round, err := buildRewardsRound(activeOnly)
		assert.Equal(t, err, nil)
		assert.Equal(t, round.Length.Uint64(), uint64(3))
		assert.Equal(t, round.Step.Cmp(new(big.Int).Mul(big.NewInt(10000), params.Energi_bn)), 0)

		assert.Equal(t, len(round.RewardsLine), 3)
		assert.Equal(t, round.RewardsLine[0].start.Uint64(), uint64(0))
		assert.Equal(t, round.RewardsLine[0].size.Cmp(masternodes[0].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[0].masternode, masternodes[0])

		assert.Equal(t, round.RewardsLine[1].start.Cmp(masternodes[0].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[1].size.Cmp(masternodes[1].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[1].masternode, masternodes[1])

		assert.Equal(t, round.RewardsLine[2].start.Cmp(new(big.Int).Add(round.RewardsLine[1].start, round.RewardsLine[1].size)), 0)
		assert.Equal(t, round.RewardsLine[2].size.Cmp(masternodes[2].CollateralAmount), 0)
		assert.Equal(t, round.RewardsLine[2].masternode, masternodes[2])

		winner, err := FindWinner(masternodes, block_i)
		assert.Equal(t, err, nil)
		assert.Equal(t, winner, masternodes[1]) // 25 % 3 = 1, 1 step
	}

	// Test 26-29 blocks
	test_winner_is(t, masternodes, 26, masternodes[2]) // 2 step
	test_winner_is(t, masternodes, 27, masternodes[0]) // 0 step
	test_winner_is(t, masternodes, 28, masternodes[1]) // 1 step
	test_winner_is(t, masternodes, 29, masternodes[2]) // 2 step
}

func Test_calcRewardPoint(t *testing.T) {
	var round RewardsRound
	// Test one-step round
	round.Step = big.NewInt(1)
	round.Length = big.NewInt(10)
	for i := 0; i < 100; i++ {
		point := calcRewardPoint(&round, big.NewInt(int64(i)))
		assert.Equal(t, int(point.Uint64()), i % 10)
	}

	// Test 10-step round bounds
	round.Step = big.NewInt(10)
	round.Length = big.NewInt(20)
	for i := 0; i < 1000; i++ {
		point := calcRewardPoint(&round, big.NewInt(int64(i)))
		assert.Equal(t, int(point.Uint64()) >= (i % 20) * 10, true)
		assert.Equal(t, int(point.Uint64()) < (i % 20) * 10 + 10, true)
	}

	// Test the equiprobability of the distribution
	round.Step = big.NewInt(10)
	round.Length = big.NewInt(3)

	pointsHits := make(map[uint64]int) // point -> number of occurrences
	for i := 0; i < 10000; i++ {
		point := calcRewardPoint(&round, big.NewInt(int64(i)))
		_, ok := pointsHits[point.Uint64()]
		if !ok {
			pointsHits[point.Uint64()] = 0
		}
		pointsHits[point.Uint64()] += 1
	}
	for _, hits := range pointsHits {
		fmt.Printf("Test_calcRewardPoint hits: %d \n", hits)
		assert.Equal(t, hits > 300 - 60, true)
		assert.Equal(t, hits < 300 + 60, true)
	}

	// Test specific 1e+32-step round
	round.Step = new(big.Int).Mul(big.NewInt(1e+16), big.NewInt(1e+16))
	round.Length = big.NewInt(20)
	point0 := calcRewardPoint(&round, big.NewInt(0))
	point5 := calcRewardPoint(&round, big.NewInt(5))
	point19 := calcRewardPoint(&round, big.NewInt(19))
	point20 := calcRewardPoint(&round, big.NewInt(20))
	point25 := calcRewardPoint(&round, big.NewInt(25))
	point39 := calcRewardPoint(&round, big.NewInt(39))

	fmt.Printf("Test_calcRewardPoint actual num: %s \n", point0.String())
	fmt.Printf("Test_calcRewardPoint actual num: %s \n", point5.String())
	fmt.Printf("Test_calcRewardPoint actual num: %s \n", point19.String())
	fmt.Printf("Test_calcRewardPoint actual num: %s \n", point20.String())
	fmt.Printf("Test_calcRewardPoint actual num: %s \n", point25.String())
	fmt.Printf("Test_calcRewardPoint actual num: %s \n", point39.String())

	point0_want,  _ := new(big.Int).SetString(  "48198034993379397001115665086549", 10)
	point5_want,  _ := new(big.Int).SetString( "548198034993379397001115665086549", 10)
	point19_want, _ := new(big.Int).SetString("1948198034993379397001115665086549", 10)
	point20_want, _ := new(big.Int).SetString(  "92190392920402856263689962707065", 10)
	point25_want, _ := new(big.Int).SetString( "592190392920402856263689962707065", 10)
	point39_want, _ := new(big.Int).SetString("1992190392920402856263689962707065", 10)

	assert.Equal(t, point0.Cmp(point0_want), 0)
	assert.Equal(t, point5.Cmp(point5_want), 0)
	assert.Equal(t, point19.Cmp(point19_want), 0)
	assert.Equal(t, point20.Cmp(point20_want), 0)
	assert.Equal(t, point25.Cmp(point25_want), 0)
	assert.Equal(t, point39.Cmp(point39_want), 0)
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