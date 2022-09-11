package handler

import (
	"encoding/json"
	"errors"
	"gacha-simulator/model"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-oauth2/oauth2/v4"
	"gorm.io/gorm"
)

func GetGameTitles(c *gin.Context) {
	var gameTitles []model.GameTitle
	if err := model.DB.
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
	var tiers []model.Tier
	if err := model.DB.
		Joins("JOIN game_titles on game_titles.id=tiers.game_title_id").
		Where("game_titles.slug = ?", gameTitleSlug).
		Preload("Translations").
		Find(&tiers).
		Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	preferred := getPreferredLanguage(c)
	for i := 0; i < len(tiers); i++ {
		j := getTranslationIndex(preferred, tiers[i])
		tiers[i].TierTranslatable = tiers[i].Translations[j].TierTranslatable
	}
	c.JSON(http.StatusOK, &tiers)
}

func GetItems(c *gin.Context) {
	gameTitleSlug := c.Param("gameTitleSlug")
	name := c.Query("name")
	var items []model.Item
	if err := model.DB.
		Joins("JOIN item_translations on item_translations.item_id=items.id").
		Joins("JOIN tiers on tiers.id=items.tier_id").
		Joins("JOIN game_titles on game_titles.id=tiers.game_title_id").
		Where("game_titles.slug", gameTitleSlug).
		Where("item_translations.name LIKE ? OR item_translations.short_name LIKE ?", "%"+name+"%", "%"+name+"%").
		Preload("Tier.Translations").
		Preload("Translations").
		Distinct().
		Find(&items).
		Limit(100).
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
		Joins("JOIN items on items.id=policies.pity_item_id").
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
		if policies[i].Pity {
			k := getTranslationIndex(preferred, policies[i].PityItem)
			policies[i].PityItem.ItemTranslatable = policies[i].PityItem.Translations[k].ItemTranslatable
			l := getTranslationIndex(preferred, policies[i].PityItem.Tier)
			policies[i].PityItem.Tier.TierTranslatable = policies[i].PityItem.Tier.Translations[l].TierTranslatable
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
		if plans[i].ItemGoals {
			var itemNumberMap map[uint]uint
			if err := json.Unmarshal([]byte(plans[i].WantedItemsJSON.String()), &itemNumberMap); err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
			plans[i].WantedItems = make([]model.ItemWithNumber, len(itemNumberMap))
			j := 0
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
					c.Status(http.StatusInternalServerError)
					return
				}
				item.Number = itemNumber
				k := getTranslationIndex(preferred, item)
				item.ItemTranslatable = item.Translations[k].ItemTranslatable
				l := getTranslationIndex(preferred, item.Tier)
				item.Tier.TierTranslatable = item.Tier.Translations[l].TierTranslatable
				plans[i].WantedItems[j] = item
				j++
			}
		}
		if plans[i].TierGoals {
			var tierNumberMap map[uint]uint
			if err := json.Unmarshal([]byte(plans[i].WantedTiersJSON.String()), &tierNumberMap); err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
			plans[i].WantedTiers = make([]model.TierWithNumber, len(tierNumberMap))
			j := 0
			for tierID, tierNumber := range tierNumberMap {
				var tier model.TierWithNumber
				if err := model.DB.
					Model(&model.Tier{}).
					Joins("JOIN game_titles on game_titles.id=tiers.game_title_id").
					Where("game_titles.slug = ?", gameTitleSlug).
					Preload("Translations").
					First(&tier, "tiers.id=?", tierID).
					Error; err != nil {
					c.Status(http.StatusInternalServerError)
					return
				}
				tier.Number = tierNumber
				k := getTranslationIndex(preferred, tier)
				tier.TierTranslatable = tier.Translations[k].TierTranslatable
				plans[i].WantedTiers[j] = tier
				j++
			}
		}
	}
	c.JSON(http.StatusOK, &plans)
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
			Order("results.time desc").
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
