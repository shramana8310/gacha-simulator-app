package gacha

import (
	"testing"
)

type RandomNumberGeneratorMock struct {
	returnValues []int
}

func (mock *RandomNumberGeneratorMock) Intn(int) int {
	r := mock.returnValues[0]
	mock.returnValues = mock.returnValues[1:]
	return r
}

func TestExecute(t *testing.T) {
	rng = &RandomNumberGeneratorMock{returnValues: []int{0, 0, 8, 2, 9, 0}}
	pityItem := Item{
		ID:    3,
		Ratio: 1,
	}
	res, err := Execute(Request{
		Tiers: []Tier{
			{
				ID:    1,
				Ratio: 9,
				Items: []Item{
					{
						ID:    1,
						Ratio: 2,
					},
					{
						ID:    2,
						Ratio: 1,
					},
				},
			},
			{
				ID:    2,
				Ratio: 1,
				Items: []Item{pityItem},
			},
		},
		ItemsIncluded: true,
		Pricing: Pricing{
			PricePerGacha:           100,
			Discount:                true,
			DiscountTrigger:         3,
			DiscountedPricePerGacha: 90,
		},
		Policies: Policies{
			Pity:        true,
			PityTrigger: 3,
			PityItem:    &pityItem,
		},
		Plan: Plan{
			Budget:               500,
			MaxConsecutiveGachas: 4,
			ItemGoals:            true,
			WantedItems: map[uint]int{
				3: 2,
			},
		},
	})
	if err != nil {
		t.Error("Unexpected error")
	}
	if res.GoalsAchieved != true {
		t.Error("Unexpected GoalsAchieved value")
	}
	if res.MoneySpent != 90*3+100*1 {
		t.Error("Unexpected MoneySpent value")
	}
	if res.Items[0].ID != 1 || res.Items[1].ID != 2 || res.Items[2].ID != 3 || res.Items[3].ID != 3 {
		t.Error("Unexpected Items")
	}
}
