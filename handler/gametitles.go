package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"gacha-simulator/model"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-oauth2/oauth2/v4"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func GetGameTitles(c *gin.Context) {
	gameTitlesModel, err := getGameTitlesModel()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	gameTitles := mapGameTitles(gameTitlesModel, c)
	c.JSON(http.StatusOK, &gameTitles)
}

func GetGameTitle(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	gameTitleModel, err := getGameTitleModel(gameTitleSlug)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if gameTitleModel == nil {
		c.Status(http.StatusNotFound)
		return
	}
	gameTitle := mapGameTitle(*gameTitleModel, c)
	c.JSON(http.StatusOK, &gameTitle)
}

func GetTiers(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	tiersModel, err := getTiersModel(gameTitleSlug)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	tiers := mapTiers(tiersModel, c)
	c.JSON(http.StatusOK, &tiers)
}

const ItemSearchLimit = 50

type ByTierAndShortName []Item

func (a ByTierAndShortName) Len() int {
	return len(a)
}
func (a ByTierAndShortName) Less(i, j int) bool {
	if a[i].Tier.ID == a[j].Tier.ID {
		return a[i].ShortName < a[j].ShortName
	} else {
		return a[i].Tier.Ratio < a[j].Tier.Ratio
	}
}
func (a ByTierAndShortName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func GetItems(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	name := c.Query("name")
	itemsModel, err := getItemsModel(gameTitleSlug, name)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	items := mapItems(itemsModel, c)
	sort.Sort(ByTierAndShortName(items))
	c.JSON(http.StatusOK, &items)
}

func GetPricings(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	pricingsModel, err := getPricingsModel(gameTitleSlug)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	pricings := mapPricings(pricingsModel, c)
	c.JSON(http.StatusOK, &pricings)
}

func GetPolicies(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	policiesModel, err := getPoliciesModel(gameTitleSlug)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	policies := mapPolicies(policiesModel, c)
	c.JSON(http.StatusOK, &policies)
}

func GetPlans(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	plansModel, err := getPlansModel(gameTitleSlug)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	plans, err := mapPlans(plansModel, c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, &plans)
}

type PresetsResponse struct {
	Tiers   []Tier   `json:"tiers"`
	Presets []Preset `json:"presets"`
}

func GetPresets(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	tiersModel, err := getTiersModel(gameTitleSlug)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	tiers := mapTiers(tiersModel, c)
	presetsModel, err := getPresetsModel(gameTitleSlug)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	presets, err := mapPresets(presetsModel, c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, &PresetsResponse{
		Tiers:   tiers,
		Presets: presets,
	})
}

type Pagination struct {
	Index        int         `json:"index"`
	Count        int         `json:"count"`
	CountPerPage int         `json:"countPerPage"`
	Total        int64       `json:"total"`
	PageIndex    int         `json:"pageIndex"`
	PageTotal    int         `json:"pageTotal"`
	Data         interface{} `json:"data"`
}

func GameTitleGachasByUser(results *[]model.Result, gameTitleSlug, userID string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Model(results).
			Joins("JOIN game_titles on game_titles.id=results.game_title_id").
			Where("game_titles.slug = ? AND results.user_id = ?", gameTitleSlug, userID).
			Preload("GameTitle.Translations")
	}
}

func GetGachas(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	accessToken, _ := c.Get("access_token")
	userID := accessToken.(oauth2.TokenInfo).GetUserID()
	pageIndexStr := c.Query("pageIndex")
	if len(pageIndexStr) == 0 {
		pageIndexStr = "0"
	}
	pageIndex, err := strconv.Atoi(pageIndexStr)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	count := 10
	offset := pageIndex * count
	var total int64

	var results []model.Result
	if err := model.DB.
		Scopes(GameTitleGachasByUser(&results, gameTitleSlug, userID)).
		Count(&total).
		Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if err := model.DB.
		Scopes(GameTitleGachasByUser(&results, gameTitleSlug, userID)).
		Order("results.time desc").
		Offset(offset).
		Limit(count).
		Omit("Items", "Request").
		Find(&results).
		Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	preferred := getPreferredLanguage(c)
	for i := 0; i < len(results); i++ {
		j := getTranslationIndex(preferred, results[i].GameTitle)
		results[i].GameTitle.GameTitleTranslatable = results[i].GameTitle.Translations[j].GameTitleTranslatable
	}
	c.JSON(http.StatusOK, Pagination{
		Index:        pageIndex * count,
		Count:        len(results),
		CountPerPage: count,
		Total:        total,
		PageIndex:    pageIndex,
		PageTotal:    int(math.Ceil(float64(total) / float64(count))),
		Data:         results,
	})
}

func getGameTitlesModel() ([]model.GameTitle, error) {
	var gameTitlesModel []model.GameTitle
	if err := model.DB.
		Order("display_order").
		Preload("Translations").
		Find(&gameTitlesModel).
		Error; err != nil {
		return nil, err
	}
	return gameTitlesModel, nil
}

func getGameTitleModel(gameTitleSlug string) (*model.GameTitle, error) {
	var gameTitleModel model.GameTitle
	if err := model.DB.
		Where("slug = ?", gameTitleSlug).
		Preload("Translations").
		Find(&gameTitleModel).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &gameTitleModel, nil
}

func getTiersModel(gameTitleSlug string) ([]model.Tier, error) {
	var tiersModel []model.Tier
	if err := model.DB.
		Joins("JOIN game_titles on game_titles.id=tiers.game_title_id").
		Where("game_titles.slug = ?", gameTitleSlug).
		Preload("Translations").
		Find(&tiersModel).
		Error; err != nil {
		return nil, err
	}
	return tiersModel, nil
}

func getTierModel(tierID uint) (*model.Tier, error) {
	var tierModel model.Tier
	if err := model.DB.
		Model(&model.Tier{}).
		Joins("JOIN game_titles on game_titles.id=tiers.game_title_id").
		Preload("Translations").
		First(&tierModel, "tiers.id=?", tierID).
		Error; err != nil {
		return nil, err
	}
	return &tierModel, nil
}

func getItemsModel(gameTitleSlug string, name string) ([]model.Item, error) {
	nameOperand := fmt.Sprintf("%%%s%%", strings.TrimSpace(strings.ToLower(name)))
	var itemsModel []model.Item
	if err := model.DB.
		Joins("JOIN item_translations on item_translations.item_id=items.id").
		Joins("JOIN tiers on tiers.id=items.tier_id").
		Joins("JOIN game_titles on game_titles.id=tiers.game_title_id").
		Where("game_titles.slug", gameTitleSlug).
		Where("lower(item_translations.name) LIKE ? OR lower(item_translations.short_name) LIKE ?", nameOperand, nameOperand).
		Preload("Tier.Translations").
		Preload("Translations").
		Distinct().
		Limit(ItemSearchLimit).
		Find(&itemsModel).
		Error; err != nil {
		return nil, err
	}
	return itemsModel, nil
}

func getItemModel(itemID uint) (*model.Item, error) {
	var itemModel model.Item
	if err := model.DB.
		Model(&model.Item{}).
		Joins("JOIN tiers on tiers.id=items.tier_id").
		Joins("JOIN game_titles on game_titles.id=tiers.game_title_id").
		Preload("Tier.Translations").
		Preload("Translations").
		First(&itemModel, "items.id=?", itemID).
		Error; err != nil {
		return nil, err
	}
	return &itemModel, nil
}

func getPricingsModel(gameTitleSlug string) ([]model.Pricing, error) {
	var pricingsModel []model.Pricing
	if err := model.DB.
		Joins("JOIN game_titles on game_titles.id=pricings.game_title_id").
		Where("game_titles.slug = ?", gameTitleSlug).
		Preload("Translations").
		Find(&pricingsModel).
		Error; err != nil {
		return nil, err
	}
	return pricingsModel, nil
}

func getPoliciesModel(gameTitleSlug string) ([]model.Policies, error) {
	var policiesModel []model.Policies
	if err := model.DB.
		Joins("LEFT JOIN items on items.id=policies.pity_item_id").
		Joins("JOIN game_titles on game_titles.id=policies.game_title_id").
		Where("game_titles.slug = ?", gameTitleSlug).
		Preload("PityItem.Tier.Translations").
		Preload("PityItem.Translations").
		Preload("Translations").
		Find(&policiesModel).
		Error; err != nil {
		return nil, err
	}
	return policiesModel, nil
}

func getPlansModel(gameTitleSlug string) ([]model.Plan, error) {
	var plansModel []model.Plan
	if err := model.DB.
		Joins("JOIN game_titles on game_titles.id=plans.game_title_id").
		Where("game_titles.slug = ?", gameTitleSlug).
		Preload("Translations").
		Find(&plansModel).
		Error; err != nil {
		return nil, err
	}
	return plansModel, nil
}

func getPresetsModel(gameTitleSlug string) ([]model.Preset, error) {
	var presetsModel []model.Preset
	if err := model.DB.
		Joins("JOIN game_titles on game_titles.id=presets.game_title_id").
		Where("game_titles.slug = ?", gameTitleSlug).
		Preload("Pricing.Translations").
		Preload("Pricing").
		Preload("Policies.PityItem.Tier.Translations").
		Preload("Policies.PityItem.Translations").
		Preload("Policies.Translations").
		Preload("Policies").
		Preload("Plan.Translations").
		Preload("Plan").
		Preload("Translations").
		Find(&presetsModel).
		Error; err != nil {
		return nil, err
	}
	return presetsModel, nil
}

func mapGameTitle(gameTitleModel model.GameTitle, c *gin.Context) *GameTitle {
	preferred := getPreferredLanguage(c)
	i := getTranslationIndex(preferred, gameTitleModel)
	return &GameTitle{
		ID:           gameTitleModel.ID,
		Slug:         gameTitleModel.Slug,
		ImageURL:     gameTitleModel.ImageURL,
		DisplayOrder: gameTitleModel.DisplayOrder,
		Name:         gameTitleModel.Translations[i].Name,
		ShortName:    gameTitleModel.Translations[i].ShortName,
		Description:  gameTitleModel.Translations[i].Description,
	}
}

func mapGameTitles(gameTitlesModel []model.GameTitle, c *gin.Context) []GameTitle {
	gameTitles := make([]GameTitle, 0)
	for i := 0; i < len(gameTitlesModel); i++ {
		gameTitle := mapGameTitle(gameTitlesModel[i], c)
		gameTitles = append(gameTitles, *gameTitle)
	}
	return gameTitles
}

func mapTier(tierModel model.Tier, c *gin.Context) *Tier {
	preferred := getPreferredLanguage(c)
	i := getTranslationIndex(preferred, tierModel)
	return &Tier{
		ID:        tierModel.ID,
		Ratio:     tierModel.Ratio,
		ImageURL:  tierModel.ImageURL,
		Name:      tierModel.Translations[i].Name,
		ShortName: tierModel.Translations[i].ShortName,
	}
}

func mapTiers(tiersModel []model.Tier, c *gin.Context) []Tier {
	tiers := make([]Tier, 0)
	for i := 0; i < len(tiersModel); i++ {
		tier := mapTier(tiersModel[i], c)
		tiers = append(tiers, *tier)
	}
	return tiers
}

func mapItem(itemModel model.Item, c *gin.Context) *Item {
	preferred := getPreferredLanguage(c)
	i := getTranslationIndex(preferred, itemModel)
	tier := mapTier(*itemModel.Tier, c)
	return &Item{
		ID:        itemModel.ID,
		Ratio:     itemModel.Ratio,
		ImageURL:  itemModel.ImageURL,
		Tier:      tier,
		Name:      itemModel.Translations[i].Name,
		ShortName: itemModel.Translations[i].ShortName,
	}
}

func mapItems(itemsModel []model.Item, c *gin.Context) []Item {
	items := make([]Item, 0)
	for i := 0; i < len(itemsModel); i++ {
		item := mapItem(itemsModel[i], c)
		items = append(items, *item)
	}
	return items
}

func mapPricing(pricingModel model.Pricing, c *gin.Context) *Pricing {
	preferred := getPreferredLanguage(c)
	i := getTranslationIndex(preferred, pricingModel)
	return &Pricing{
		ID:                      pricingModel.ID,
		PricePerGacha:           pricingModel.PricePerGacha,
		Discount:                pricingModel.Discount,
		DiscountTrigger:         pricingModel.DiscountTrigger,
		DiscountedPricePerGacha: pricingModel.DiscountedPricePerGacha,
		Name:                    pricingModel.Translations[i].Name,
	}
}

func mapPricings(pricingsModel []model.Pricing, c *gin.Context) []Pricing {
	pricings := make([]Pricing, 0)
	for i := 0; i < len(pricingsModel); i++ {
		pricing := mapPricing(pricingsModel[i], c)
		pricings = append(pricings, *pricing)
	}
	return pricings
}

func mapPolicy(policyModel model.Policies, c *gin.Context) *Policies {
	preferred := getPreferredLanguage(c)
	i := getTranslationIndex(preferred, policyModel)
	var pityItem *Item
	if policyModel.Pity && policyModel.PityItem != nil {
		pityItem = mapItem(*policyModel.PityItem, c)
	}
	return &Policies{
		ID:          policyModel.ID,
		Pity:        policyModel.Pity,
		PityTrigger: policyModel.PityTrigger,
		PityItem:    pityItem,
		Name:        policyModel.Translations[i].Name,
	}
}

func mapPolicies(policiesModel []model.Policies, c *gin.Context) []Policies {
	policies := make([]Policies, 0)
	for i := 0; i < len(policiesModel); i++ {
		policy := mapPolicy(policiesModel[i], c)
		policies = append(policies, *policy)
	}
	return policies
}

func mapPlan(planModel model.Plan, c *gin.Context) (*Plan, error) {
	preferred := getPreferredLanguage(c)
	i := getTranslationIndex(preferred, planModel)
	wantedItems, err := mapWantedItems(planModel, c)
	if err != nil {
		return nil, err
	}
	wantedTiers, err := mapWantedTiers(planModel, c)
	if err != nil {
		return nil, err
	}
	return &Plan{
		ID:                   planModel.ID,
		Budget:               planModel.Budget,
		MaxConsecutiveGachas: planModel.MaxConsecutiveGachas,
		ItemGoals:            planModel.ItemGoals,
		WantedItems:          wantedItems,
		TierGoals:            planModel.TierGoals,
		WantedTiers:          wantedTiers,
		Name:                 planModel.Translations[i].Name,
	}, nil
}

func mapPlans(plansModel []model.Plan, c *gin.Context) ([]Plan, error) {
	plans := make([]Plan, 0)
	for i := 0; i < len(plansModel); i++ {
		plan, err := mapPlan(plansModel[i], c)
		if err != nil {
			return nil, err
		}
		plans = append(plans, *plan)
	}
	return plans, nil
}

func mapWantedItems(planModel model.Plan, c *gin.Context) ([]ItemWithNumber, error) {
	wantedItems := make([]ItemWithNumber, 0)
	if planModel.ItemGoals {
		itemNumberMap, err := toMap(planModel.WantedItemsJSON)
		if err != nil {
			return nil, err
		}
		for itemID, itemNumber := range itemNumberMap {
			itemModel, err := getItemModel(itemID)
			if err != nil {
				return nil, err
			}
			item := mapItem(*itemModel, c)
			wantedItems = append(wantedItems, ItemWithNumber{
				Item:   *item,
				Number: itemNumber,
			})
		}
	}
	return wantedItems, nil
}

func mapWantedTiers(planModel model.Plan, c *gin.Context) ([]TierWithNumber, error) {
	wantedTiers := make([]TierWithNumber, 0)
	if planModel.TierGoals {
		tierNumberMap, err := toMap(planModel.WantedTiersJSON)
		if err != nil {
			return nil, err
		}
		for tierID, tierNumber := range tierNumberMap {
			tierModel, err := getTierModel(tierID)
			if err != nil {
				return nil, err
			}
			tier := mapTier(*tierModel, c)
			wantedTiers = append(wantedTiers, TierWithNumber{
				Tier:   *tier,
				Number: tierNumber,
			})
		}
	}
	return wantedTiers, nil
}

func mapPreset(presetModel model.Preset, c *gin.Context) (*Preset, error) {
	preferred := getPreferredLanguage(c)
	i := getTranslationIndex(preferred, presetModel)
	var pricing *Pricing
	if presetModel.Pricing != nil {
		pricing = mapPricing(*presetModel.Pricing, c)
	}
	var policies *Policies
	if presetModel.Policies != nil {
		policies = mapPolicy(*presetModel.Policies, c)
	}
	var plan *Plan
	if presetModel.Plan != nil {
		planMapped, err := mapPlan(*presetModel.Plan, c)
		if err != nil {
			return nil, err
		}
		plan = planMapped
	}
	return &Preset{
		ID:          presetModel.ID,
		Pricing:     pricing,
		Policies:    policies,
		Plan:        plan,
		Name:        presetModel.Translations[i].Name,
		Description: presetModel.Translations[i].Description,
	}, nil
}

func mapPresets(presetsModel []model.Preset, c *gin.Context) ([]Preset, error) {
	presets := make([]Preset, 0)
	for i := 0; i < len(presetsModel); i++ {
		preset, err := mapPreset(presetsModel[i], c)
		if err != nil {
			return nil, err
		}
		presets = append(presets, *preset)
	}
	return presets, nil
}

func toMap(jsonData datatypes.JSON) (map[uint]uint, error) {
	var wantedThingsMap map[uint]uint
	if err := json.Unmarshal([]byte(jsonData.String()), &wantedThingsMap); err != nil {
		return nil, err
	}
	return wantedThingsMap, nil
}
