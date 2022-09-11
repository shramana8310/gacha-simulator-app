package handler

import (
	"encoding/json"
	"errors"
	"gacha-simulator/gacha"
	"gacha-simulator/model"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-oauth2/oauth2/v4"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func mapRequest(gachaRequest model.GachaRequest) gacha.Request {
	var tiers []gacha.Tier
	for _, tier := range gachaRequest.Tiers {
		var items []gacha.Item
		for _, item := range tier.Items {
			items = append(items, gacha.Item{
				ID:    item.ID,
				Ratio: item.Ratio,
			})
		}
		tiers = append(tiers, gacha.Tier{
			ID:    tier.ID,
			Ratio: tier.Ratio,
			Items: items,
		})
	}
	wantedItems := make(map[uint]int)
	if gachaRequest.Plan.ItemGoals {
		for _, wantedItem := range gachaRequest.Plan.WantedItems {
			wantedItems[wantedItem.ID] = int(wantedItem.Number)
		}
	}
	wantedTiers := make(map[uint]int)
	if gachaRequest.Plan.TierGoals {
		for _, wantedTier := range gachaRequest.Plan.WantedTiers {
			wantedTiers[wantedTier.ID] = int(wantedTier.Number)
		}
	}
	var pityItem gacha.Item
	if gachaRequest.Policies.Pity && gachaRequest.Policies.PityItem != nil {
		pityItem = gacha.Item{ID: gachaRequest.Policies.PityItem.ID}
	}
	return gacha.Request{
		Tiers:         tiers,
		ItemsIncluded: gachaRequest.ItemsIncluded,
		Pricing: gacha.Pricing{
			PricePerGacha:           gachaRequest.Pricing.PricePerGacha,
			Discount:                gachaRequest.Pricing.Discount,
			DiscountTrigger:         gachaRequest.Pricing.DiscountTrigger,
			DiscountedPricePerGacha: gachaRequest.Pricing.DiscountedPricePerGacha,
		},
		Policies: gacha.Policies{
			Pity:        gachaRequest.Policies.Pity,
			PityTrigger: gachaRequest.Policies.PityTrigger,
			PityItem:    &pityItem,
		},
		Plan: gacha.Plan{
			Budget:               gachaRequest.Plan.Budget,
			MaxConsecutiveGachas: gachaRequest.Plan.MaxConsecutiveGachas,
			ItemGoals:            gachaRequest.Plan.ItemGoals,
			WantedItems:          wantedItems,
			TierGoals:            gachaRequest.Plan.TierGoals,
			WantedTiers:          wantedTiers,
		},
		GetItemCount: func(tierID uint) (int64, error) {
			var count int64
			if err := model.DB.
				Model(&model.Item{}).
				Where("tier_id", tierID).
				Count(&count).
				Error; err != nil {
				return -1, err
			}
			return count, nil
		},
		GetItemFromIndex: func(tierID uint, index int) (gacha.Item, error) {
			var item model.Item
			if err := model.DB.
				Model(&model.Item{}).
				Where("tier_id", tierID).
				Offset(index).
				Preload("Tier").
				First(&item).
				Error; err != nil {
				return gacha.Item{}, err
			}
			return gacha.Item{
				ID:   item.ID,
				Tier: &gacha.Tier{ID: item.Tier.ID},
			}, nil
		},
		GetItemFromID: func(itemID uint) (gacha.Item, error) {
			var item model.Item
			if err := model.DB.
				Preload("Tier").
				First(&item, "items.id=?", itemID).
				Error; err != nil {
				return gacha.Item{}, nil
			}
			return gacha.Item{
				ID:   item.ID,
				Tier: &gacha.Tier{ID: item.Tier.ID},
			}, nil
		},
		GetItemCountFromIDs: func(itemIDs []uint) (int64, error) {
			var count int64
			if err := model.DB.
				Model(&model.Item{}).
				Where("id IN ?", itemIDs).
				Count(&count).
				Error; err != nil {
				return -1, err
			}
			return count, nil
		},
		GetTierCountFromIDs: func(tierIDs []uint) (int64, error) {
			var count int64
			if err := model.DB.
				Model(&model.Tier{}).
				Where("id IN ?", tierIDs).
				Count(&count).
				Error; err != nil {
				return -1, err
			}
			return count, nil
		},
	}
}

func PostGachas(c *gin.Context) {
	preferred := getPreferredLanguage(c)
	var gachaRequest model.GachaRequest
	c.Bind(&gachaRequest)

	var gameTitle model.GameTitle
	if err := model.DB.
		Preload("Translations").
		First(&gameTitle, gachaRequest.GameTitle.ID).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(http.StatusBadRequest)
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}
	i := getTranslationIndex(preferred, gameTitle)
	gameTitle.GameTitleTranslatable = gameTitle.Translations[i].GameTitleTranslatable

	request := mapRequest(gachaRequest)
	if err := gacha.Validate(request); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	result, err := gacha.Execute(request)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	itemIDSet := make(map[uint]bool)
	itemIDs := make([]uint, len(result.Items))
	for i, item := range result.Items {
		itemIDSet[item.ID] = true
		itemIDs[i] = item.ID
	}
	itemIDsUnique := make([]uint, 0)
	items := make([]model.Item, 0)
	for itemID := range itemIDSet {
		itemIDsUnique = append(itemIDsUnique, itemID)
	}
	if err := model.DB.
		Model(&model.Item{}).
		Preload("Tier.Translations").
		Preload("Translations").
		Where("id IN ?", itemIDsUnique).
		Find(&items).
		Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	for i := 0; i < len(items); i++ {
		j := getTranslationIndex(preferred, items[i])
		items[i].ItemTranslatable = items[i].Translations[j].ItemTranslatable
		k := getTranslationIndex(preferred, items[i].Tier)
		items[i].Tier.TierTranslatable = items[i].Tier.Translations[k].TierTranslatable
	}

	itemIDsJSON, _ := json.Marshal(itemIDs)
	itemsJSON, _ := json.Marshal(items)
	requestJSON, _ := json.Marshal(request)
	now := time.Now()

	accessToken, _ := c.Get("access_token")
	userID := accessToken.(oauth2.TokenInfo).GetUserID()

	resultModel := model.Result{
		Request:       datatypes.JSON(requestJSON),
		ItemIDs:       datatypes.JSON(itemIDsJSON),
		Items:         datatypes.JSON(itemsJSON),
		ItemsResponse: items,
		GoalsAchieved: result.GoalsAchieved,
		MoneySpent:    result.MoneySpent,
		Time:          now,
		GameTitle:     &gameTitle,
		UserID:        userID,
		Public:        false,
	}
	if err := model.DB.Create(&resultModel).Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, resultModel)
}

func GetGacha(c *gin.Context) {
	resultID := c.Param("resultID")
	var result model.Result
	if err := model.DB.
		Preload("GameTitle.Translations").
		First(&result, resultID).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(http.StatusNotFound)
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}
	accessToken, _ := c.Get("access_token")
	userID := accessToken.(oauth2.TokenInfo).GetUserID()
	if !result.Public && result.UserID != userID {
		c.Status(http.StatusForbidden)
		return
	}

	preferred := getPreferredLanguage(c)
	i := getTranslationIndex(preferred, result.GameTitle)
	result.GameTitle.GameTitleTranslatable = result.GameTitle.Translations[i].GameTitleTranslatable
	itemsJSON, _ := result.Items.MarshalJSON()
	json.Unmarshal(itemsJSON, &result.ItemsResponse)

	c.JSON(http.StatusOK, result)
}

type PatchGachaRequest struct {
	Public bool `json:"public"`
}

func PatchGacha(c *gin.Context) {
	resultID := c.Param("resultID")
	var patchGachaRequest PatchGachaRequest
	c.Bind(&patchGachaRequest)
	accessToken, _ := c.Get("access_token")
	userID := accessToken.(oauth2.TokenInfo).GetUserID()

	var result model.Result
	if err := model.DB.
		Model(&model.Result{}).
		Where("id = ? AND user_id = ?", resultID, userID).
		First(&result).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(http.StatusNotFound)
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := model.DB.
		Model(&model.Result{}).
		Where("id = ? AND user_id = ?", resultID, userID).
		Update("public", patchGachaRequest.Public).
		Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

func DeleteGacha(c *gin.Context) {
	resultID := c.Param("resultID")
	accessToken, _ := c.Get("access_token")
	userID := accessToken.(oauth2.TokenInfo).GetUserID()

	var result model.Result
	if err := model.DB.
		Model(&model.Result{}).
		Where("id = ? AND user_id = ?", resultID, userID).
		First(&result).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(http.StatusNotFound)
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := model.DB.
		Where("id = ? AND user_id = ?", resultID, userID).
		Delete(&model.Result{}).
		Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}
