package gacha

import (
	"errors"
	"math/rand"
	"time"
)

type RandomNumberGenerator interface {
	Intn(n int) int
}

var rng RandomNumberGenerator

type Item struct {
	ID    uint  `json:"id"`
	Ratio int   `json:"ratio"`
	Tier  *Tier `json:"-"`
}

type Tier struct {
	ID        uint   `json:"id"`
	Ratio     int    `json:"ratio"`
	Items     []Item `json:"items"`
	ItemCount int64  `json:"-"`
}

type Ratioer interface {
	getRatio() int
}

func (item Item) getRatio() int {
	return item.Ratio
}

func (tier Tier) getRatio() int {
	return tier.Ratio
}

type Pricing struct {
	PricePerGacha           float64 `json:"pricePerGacha"`
	Discount                bool    `json:"discount"`
	DiscountTrigger         int     `json:"discountTrigger"`
	DiscountedPricePerGacha float64 `json:"discountedPricePerGacha"`
}

type Policies struct {
	Pity        bool  `json:"pity"`
	PityTrigger int   `json:"pityTrigger"`
	PityItem    *Item `json:"pityItem"`
}

type Plan struct {
	Budget               float64      `json:"budget"`
	MaxConsecutiveGachas int          `json:"maxConsecutiveGachas"`
	ItemGoals            bool         `json:"itemGoals"`
	WantedItems          map[uint]int `json:"wantedItems"`
	TierGoals            bool         `json:"tierGoals"`
	WantedTiers          map[uint]int `json:"wantedTiers"`
}

type Request struct {
	Tiers               []Tier                                     `json:"tiers"`
	ItemsIncluded       bool                                       `json:"itemsIncluded"`
	Pricing             Pricing                                    `json:"pricing"`
	Policies            Policies                                   `json:"policies"`
	Plan                Plan                                       `json:"plan"`
	GetItemCount        func(tierID uint) (int64, error)           `json:"-"`
	GetItemFromIndex    func(tierID uint, index int) (Item, error) `json:"-"`
	GetItemFromID       func(itemID uint) (Item, error)            `json:"-"`
	GetItemCountFromIDs func(itemIDs []uint) (int64, error)        `json:"-"`
	GetTierCountFromIDs func(tierIDs []uint) (int64, error)        `json:"-"`
}

type Result struct {
	Items         []Item  `json:"items"`
	GoalsAchieved bool    `json:"goalsAchieved"`
	MoneySpent    float64 `json:"moneySpent"`
}

func Execute(request Request) (Result, error) {
	result := Result{
		Items:         make([]Item, 0),
		GoalsAchieved: false,
		MoneySpent:    0,
	}
	if err := prepareRequest(&request); err != nil {
		return result, err
	}
	getItemFromIndexCached := getItemFromIndexCachedClosure(request.GetItemFromIndex)
	var count int
	for i := 0; i < request.Plan.MaxConsecutiveGachas; i++ {
		if exceedsBudget(i+1, request.Pricing, request.Plan.Budget) {
			break
		}
		var selectedItem Item
		if shouldSelectPityItem(i+1, request.Policies, result) {
			selectedItem = *request.Policies.PityItem
		} else {
			if item, err := selectRandomItemFromRandomTier(
				request.Tiers,
				request.ItemsIncluded,
				getItemFromIndexCached,
			); err != nil {
				return result, err
			} else {
				selectedItem = item
			}
		}
		result.Items = append(result.Items, selectedItem)
		count = i + 1
		if (request.Plan.ItemGoals || request.Plan.TierGoals) && meetsGoals(result, request.Plan) {
			result.GoalsAchieved = true
			break
		}
	}
	result.MoneySpent = calculatePrice(count, request.Pricing)
	return result, nil
}

func selectRandomItemFromRandomTier(
	tiers []Tier,
	itemsIncluded bool,
	getItemFromIndex func(tierID uint, index int) (Item, error),
) (Item, error) {
	ratioers := make([]Ratioer, len(tiers))
	for i := range tiers {
		ratioers[i] = tiers[i]
	}
	selectedRatioer := selectRandomRatioer(ratioers)
	selectedTier := selectedRatioer.(Tier)
	if itemsIncluded {
		return selectRandomItem(selectedTier.Items), nil
	} else {
		r := rng.Intn(int(selectedTier.ItemCount))
		return getItemFromIndex(selectedTier.ID, r)
	}
}

func selectRandomItem(items []Item) Item {
	ratioers := make([]Ratioer, len(items))
	for i := range items {
		ratioers[i] = items[i]
	}
	selectedRatioer := selectRandomRatioer(ratioers)
	return selectedRatioer.(Item)
}

func selectRandomRatioer(ratioers []Ratioer) Ratioer {
	ratioRatioerMap := make(map[int]Ratioer)
	ratioSum := 0
	for _, ratioer := range ratioers {
		if ratioer.getRatio() > 0 {
			ratioSum += ratioer.getRatio()
			ratioRatioerMap[ratioSum] = ratioer
		}
	}
	r := rng.Intn(ratioSum)
	var selectedRatioer Ratioer
	for ratioCeiling, ratioer := range ratioRatioerMap {
		ratioBottom := ratioCeiling - ratioer.getRatio()
		if r < ratioCeiling && r >= ratioBottom {
			selectedRatioer = ratioer
			break
		}
	}
	return selectedRatioer
}

func exceedsBudget(count int, pricing Pricing, budget float64) bool {
	price := calculatePrice(count, pricing)
	return price > budget
}

func calculatePrice(count int, pricing Pricing) float64 {
	if pricing.Discount && count >= pricing.DiscountTrigger {
		dividend := count / pricing.DiscountTrigger
		remainder := count % pricing.DiscountTrigger
		return (pricing.DiscountedPricePerGacha * float64(pricing.DiscountTrigger) *
			float64(dividend)) + (pricing.PricePerGacha * float64(remainder))
	} else {
		return pricing.PricePerGacha * float64(count)
	}
}

func shouldSelectPityItem(count int, policies Policies, result Result) bool {
	if policies.Pity && count >= policies.PityTrigger {
		for _, item := range result.Items {
			if policies.PityItem.ID == item.ID {
				return false
			}
		}
		return true
	} else {
		return false
	}
}

func meetsGoals(result Result, plan Plan) bool {
	return meetsItemGoals(result, plan) && meetsTierGoals(result, plan)
}

func meetsItemGoals(result Result, plan Plan) bool {
	if plan.ItemGoals {
		for itemID, wantedCount := range plan.WantedItems {
			count := 0
			for _, item := range result.Items {
				if item.ID == itemID {
					count++
				}
				if count >= wantedCount {
					break
				}
			}
			if count < wantedCount {
				return false
			}
		}
		return true
	} else {
		return true
	}
}

func meetsTierGoals(result Result, plan Plan) bool {
	if plan.TierGoals {
		for tierID, wantedCount := range plan.WantedTiers {
			count := 0
			for _, item := range result.Items {
				if item.Tier.ID == tierID {
					count++
				}
				if count >= wantedCount {
					break
				}
			}
			if count < wantedCount {
				return false
			}
		}
		return true
	} else {
		return true
	}
}

func prepareRequest(request *Request) error {
	if request.ItemsIncluded {
		ensureItemTierReferences(request.Tiers, &request.Policies)
	} else {
		if err := countItems(request); err != nil {
			return err
		}
		if request.Policies.Pity {
			pityItem, err := request.GetItemFromID(request.Policies.PityItem.ID)
			if err != nil {
				return err
			}
			if pityItem.Tier == nil {
				return errors.New("Pity item's tier not found")
			}
			request.Policies.PityItem = &pityItem
		}
	}
	return nil
}

func ensureItemTierReferences(tiers []Tier, policies *Policies) {
	for i := 0; i < len(tiers); i++ {
		for j := 0; j < len(tiers[i].Items); j++ {
			if tiers[i].Items[j].Tier == nil {
				tiers[i].Items[j].Tier = &tiers[i]
			}
		}
	}
	if policies.Pity {
		found := false
		for i := 0; i < len(tiers); i++ {
			for j := 0; j < len(tiers[i].Items); j++ {
				if tiers[i].Items[j].ID == policies.PityItem.ID {
					policies.PityItem = &tiers[i].Items[j]
					found = true
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}
	}
}

func countItems(request *Request) error {
	for i := 0; i < len(request.Tiers); i++ {
		count, err := request.GetItemCount(request.Tiers[i].ID)
		if err != nil {
			return err
		}
		if count == 0 {
			return errors.New("Zero item count")
		}
		request.Tiers[i].ItemCount = count
	}
	return nil
}

func getItemFromIndexCachedClosure(getItemFromIndex func(uint, int) (Item, error)) func(uint, int) (Item, error) {
	tierCache := make(map[uint]map[int]Item)
	return func(tierID uint, index int) (Item, error) {
		if itemCache, ok := tierCache[tierID]; ok {
			if item, ok := itemCache[index]; ok {
				return item, nil
			}
		}
		item, err := getItemFromIndex(tierID, index)
		if err != nil {
			return Item{}, err
		}
		if _, ok := tierCache[tierID]; !ok {
			tierCache[tierID] = make(map[int]Item)
		}
		tierCache[tierID][index] = item
		return item, nil
	}
}

func Validate(request Request) error {
	if err := validateTiersAndItems(request); err != nil {
		return err
	}
	if err := validatePricing(request); err != nil {
		return err
	}
	if err := validatePolicies(request); err != nil {
		return err
	}
	if err := validatePlan(request); err != nil {
		return err
	}
	return nil
}

func validateTiersAndItems(request Request) error {
	if len(request.Tiers) == 0 {
		return errors.New("Tiers empty")
	}
	tierRatioSum := 0
	itemRatioSum := 0
	tierIDs := make([]uint, 0)
	itemIDs := make([]uint, 0)
	for _, tier := range request.Tiers {
		tierIDs = append(tierIDs, tier.ID)
		if tier.Ratio < 0 {
			return errors.New("Negative tier ratio")
		}
		tierRatioSum += tier.Ratio
		if request.ItemsIncluded {
			if len(tier.Items) == 0 {
				return errors.New("Items empty")
			}
			for _, item := range tier.Items {
				itemIDs = append(itemIDs, item.ID)
				if item.Ratio < 0 {
					return errors.New("Negative item ratio")
				}
				itemRatioSum += item.Ratio
			}
		}
	}
	tierCount, err := request.GetTierCountFromIDs(tierIDs)
	if err != nil {
		return err
	}
	if len(tierIDs) != int(tierCount) {
		return errors.New("Some tier not found")
	}
	if request.ItemsIncluded {
		itemCount, err := request.GetItemCountFromIDs(itemIDs)
		if err != nil {
			return err
		}
		if len(itemIDs) != int(itemCount) {
			return errors.New("Some item not found")
		}
	}
	if tierRatioSum == 0 {
		return errors.New("Tier ratio zero")
	}
	if request.ItemsIncluded {
		if itemRatioSum == 0 {
			return errors.New("Item ratio zero")
		}
	}
	return nil
}

func validatePricing(request Request) error {
	if request.Pricing.PricePerGacha < 0 {
		return errors.New("Negative price per gacha")
	}
	if request.Pricing.Discount {
		if request.Pricing.DiscountTrigger <= 0 {
			return errors.New("Non-positive discount trigger")
		}
		if request.Pricing.DiscountedPricePerGacha < 0 {
			return errors.New("Negative discounted price per gacha")
		}
		if request.Pricing.DiscountedPricePerGacha > request.Pricing.PricePerGacha {
			return errors.New("Discounted price per gacha greater than price per gacha")
		}
	}
	return nil
}

func validatePolicies(request Request) error {
	if request.Policies.Pity {
		if request.Policies.PityTrigger < 0 {
			return errors.New("Negative pity trigger")
		}
		if request.Policies.PityItem == nil {
			return errors.New("Pity item empty")
		}
		if _, err := request.GetItemFromID(request.Policies.PityItem.ID); err != nil {
			return errors.New("Pity item not found")
		}
	}
	return nil
}

func validatePlan(request Request) error {
	if request.Plan.Budget < 0 {
		return errors.New("Negative budget")
	}
	if request.Plan.MaxConsecutiveGachas < 0 {
		return errors.New("Negative max consecutive gachas")
	}
	if request.Plan.MaxConsecutiveGachas > 1000 {
		return errors.New("Exceeded max consecutive gacha limit")
	}
	if request.Plan.ItemGoals {
		if len(request.Plan.WantedItems) == 0 {
			return errors.New("Wanted items empty")
		}
		itemIDs := make([]uint, 0)
		for itemID, itemNumber := range request.Plan.WantedItems {
			itemIDs = append(itemIDs, itemID)
			if itemNumber < 0 {
				return errors.New("Negative wanted item number")
			}
		}
		itemCount, err := request.GetItemCountFromIDs(itemIDs)
		if err != nil {
			return err
		}
		if len(itemIDs) != int(itemCount) {
			return errors.New("Some wanted item not found")
		}
	}
	if request.Plan.TierGoals {
		if len(request.Plan.WantedTiers) == 0 {
			return errors.New("Wanted tiers empty")
		}
		tierIDs := make([]uint, 0)
		for tierID, tierNumber := range request.Plan.WantedTiers {
			tierIDs = append(tierIDs, tierID)
			if tierNumber < 0 {
				return errors.New("Negative wanted tier number")
			}
		}
		tierCount, err := request.GetTierCountFromIDs(tierIDs)
		if err != nil {
			return err
		}
		if len(tierIDs) != int(tierCount) {
			return errors.New("Some tier not found")
		}
	}
	return nil
}

func init() {
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
}
