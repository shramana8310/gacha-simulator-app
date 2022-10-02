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
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

func GetGameTitles(c *gin.Context) {
	var gameTitles []model.GameTitle
	if err := model.DB.
		Order("display_order").
		Preload("Translations").
		Find(&gameTitles).
		Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	preferred := getPreferredLanguage(c)
	for i := 0; i < len(gameTitles); i++ {
		j := getTranslationIndex(preferred, gameTitles[i])
		gameTitles[i].GameTitleTranslatable = gameTitles[i].Translations[j].GameTitleTranslatable
	}
	c.JSON(http.StatusOK, &gameTitles)
}

func GetGameTitle(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	var gameTitle model.GameTitle
	if err := model.DB.
		Where("slug = ?", gameTitleSlug).
		Preload("Translations").
		Find(&gameTitle).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(http.StatusNotFound)
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}
	preferred := getPreferredLanguage(c)
	i := getTranslationIndex(preferred, gameTitle)
	gameTitle.GameTitleTranslatable = gameTitle.Translations[i].GameTitleTranslatable
	c.JSON(http.StatusOK, &gameTitle)
}

func GetTiers(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	preferred := getPreferredLanguage(c)
	tiers, err := getTiers(gameTitleSlug, preferred)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, &tiers)
}

const ItemSearchLimit = 50

type ByTierAndShortName []model.Item

func (a ByTierAndShortName) Len() int {
	return len(a)
}
func (a ByTierAndShortName) Less(i, j int) bool {
	if a[i].TierID == a[j].TierID {
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
	whereOperand := fmt.Sprintf("%%%s%%", strings.TrimSpace(strings.ToLower(name)))
	var items []model.Item
	if err := model.DB.
		Joins("JOIN item_translations on item_translations.item_id=items.id").
		Joins("JOIN tiers on tiers.id=items.tier_id").
		Joins("JOIN game_titles on game_titles.id=tiers.game_title_id").
		Where("game_titles.slug", gameTitleSlug).
		Where("lower(item_translations.name) LIKE ? OR lower(item_translations.short_name) LIKE ?", whereOperand, whereOperand).
		Preload("Tier.Translations").
		Preload("Translations").
		Distinct().
		Limit(ItemSearchLimit).
		Find(&items).
		Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	preferred := getPreferredLanguage(c)
	for i := 0; i < len(items); i++ {
		j := getTranslationIndex(preferred, items[i])
		items[i].ItemTranslatable = items[i].Translations[j].ItemTranslatable
		k := getTranslationIndex(preferred, items[i].Tier)
		items[i].Tier.TierTranslatable = items[i].Tier.Translations[k].TierTranslatable
	}
	sort.Sort(ByTierAndShortName(items))
	c.JSON(http.StatusOK, &items)
}

func GetPricings(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	var pricings []model.Pricing
	if err := model.DB.
		Joins("JOIN game_titles on game_titles.id=pricings.game_title_id").
		Where("game_titles.slug = ?", gameTitleSlug).
		Preload("Translations").
		Find(&pricings).
		Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	preferred := getPreferredLanguage(c)
	for i := 0; i < len(pricings); i++ {
		j := getTranslationIndex(preferred, pricings[i])
		pricings[i].PricingTranslatable = pricings[i].Translations[j].PricingTranslatable
	}
	c.JSON(http.StatusOK, &pricings)
}

func GetPolicies(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	var policies []model.Policies
	if err := model.DB.
		Joins("LEFT JOIN items on items.id=policies.pity_item_id").
		Joins("JOIN game_titles on game_titles.id=policies.game_title_id").
		Where("game_titles.slug = ?", gameTitleSlug).
		Preload("PityItem.Tier.Translations").
		Preload("PityItem.Translations").
		Preload("Translations").
		Find(&policies).
		Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	preferred := getPreferredLanguage(c)
	for i := 0; i < len(policies); i++ {
		j := getTranslationIndex(preferred, policies[i])
		policies[i].PoliciesTranslatable = policies[i].Translations[j].PoliciesTranslatable
		err := complementPolicies(&policies[i], gameTitleSlug, preferred)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
	}
	c.JSON(http.StatusOK, &policies)
}

func GetPlans(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	var plans []model.Plan
	if err := model.DB.
		Joins("JOIN game_titles on game_titles.id=plans.game_title_id").
		Where("game_titles.slug = ?", gameTitleSlug).
		Preload("Translations").
		Find(&plans).
		Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	preferred := getPreferredLanguage(c)
	for i := 0; i < len(plans); i++ {
		j := getTranslationIndex(preferred, plans[i])
		plans[i].PlanTranslatable = plans[i].Translations[j].PlanTranslatable
		err := complementPlan(&plans[i], gameTitleSlug, preferred)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
	}
	c.JSON(http.StatusOK, &plans)
}

type PresetsResponse struct {
	Tiers   []model.Tier   `json:"tiers"`
	Presets []model.Preset `json:"presets"`
}

func GetPresets(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	preferred := getPreferredLanguage(c)
	tiers, err := getTiers(gameTitleSlug, preferred)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	var presets []model.Preset
	if err := model.DB.
		Joins("JOIN game_titles on game_titles.id=presets.game_title_id").
		Where("game_titles.slug = ?", gameTitleSlug).
		Preload("Pricing").
		Preload("Policies.PityItem.Tier.Translations").
		Preload("Policies.PityItem.Translations").
		Preload("Policies").
		Preload("Plan").
		Preload("Translations").
		Find(&presets).
		Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	for i := 0; i < len(presets); i++ {
		j := getTranslationIndex(preferred, presets[i])
		presets[i].PresetTranslatable = presets[i].Translations[j].PresetTranslatable
		if presets[i].Policies != nil {
			err := complementPolicies(presets[i].Policies, gameTitleSlug, preferred)
			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
		}
		if presets[i].Plan != nil {
			err := complementPlan(presets[i].Plan, gameTitleSlug, preferred)
			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
		}
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

func getTiers(gameTitleSlug string, preferred []language.Tag) ([]model.Tier, error) {
	var tiers []model.Tier
	if err := model.DB.
		Joins("JOIN game_titles on game_titles.id=tiers.game_title_id").
		Where("game_titles.slug = ?", gameTitleSlug).
		Preload("Translations").
		Find(&tiers).
		Error; err != nil {
		return nil, err
	}
	for i := 0; i < len(tiers); i++ {
		j := getTranslationIndex(preferred, tiers[i])
		tiers[i].TierTranslatable = tiers[i].Translations[j].TierTranslatable
	}
	return tiers, nil
}

func complementPlan(plan *model.Plan, gameTitleSlug string, preferred []language.Tag) error {
	if plan.ItemGoals {
		var itemNumberMap map[uint]uint
		if err := json.Unmarshal([]byte(plan.WantedItemsJSON.String()), &itemNumberMap); err != nil {
			return err
		}
		plan.WantedItems = make([]model.ItemWithNumber, len(itemNumberMap))
		i := 0
		for itemID, itemNumber := range itemNumberMap {
			var item model.ItemWithNumber
			if err := model.DB.
				Model(&model.Item{}).
				Joins("JOIN tiers on tiers.id=items.tier_id").
				Joins("JOIN game_titles on game_titles.id=tiers.game_title_id").
				Where("game_titles.slug = ?", gameTitleSlug).
				Preload("Tier.Translations").
				Preload("Translations").
				First(&item, "items.id=?", itemID).
				Error; err != nil {
				return err
			}
			item.Number = itemNumber
			j := getTranslationIndex(preferred, item)
			item.ItemTranslatable = item.Translations[j].ItemTranslatable
			k := getTranslationIndex(preferred, item.Tier)
			item.Tier.TierTranslatable = item.Tier.Translations[k].TierTranslatable
			plan.WantedItems[i] = item
			i++
		}
	}
	if plan.TierGoals {
		var tierNumberMap map[uint]uint
		if err := json.Unmarshal([]byte(plan.WantedTiersJSON.String()), &tierNumberMap); err != nil {
			return err
		}
		plan.WantedTiers = make([]model.TierWithNumber, len(tierNumberMap))
		i := 0
		for tierID, tierNumber := range tierNumberMap {
			var tier model.TierWithNumber
			if err := model.DB.
				Model(&model.Tier{}).
				Joins("JOIN game_titles on game_titles.id=tiers.game_title_id").
				Where("game_titles.slug = ?", gameTitleSlug).
				Preload("Translations").
				First(&tier, "tiers.id=?", tierID).
				Error; err != nil {
				return err
			}
			tier.Number = tierNumber
			j := getTranslationIndex(preferred, tier)
			tier.TierTranslatable = tier.Translations[j].TierTranslatable
			plan.WantedTiers[i] = tier
			i++
		}
	}
	return nil
}

func complementPolicies(policies *model.Policies, gameTitleSlug string, preferred []language.Tag) error {
	if policies.Pity && policies.PityItem != nil {
		i := getTranslationIndex(preferred, policies.PityItem)
		policies.PityItem.ItemTranslatable = policies.PityItem.Translations[i].ItemTranslatable
		j := getTranslationIndex(preferred, policies.PityItem.Tier)
		policies.PityItem.Tier.TierTranslatable = policies.PityItem.Tier.Translations[j].TierTranslatable
	}
	return nil
}
