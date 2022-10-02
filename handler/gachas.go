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

type ResultResponse struct {
	*model.Result
	Items                []model.Item `json:"items"`
	RemainingWantedItems []model.Item `json:"remainingWantedItems"`
	RemainingWantedTiers []model.Tier `json:"remainingWantedTiers"`
}

type PatchGachaRequest struct {
	Public bool `json:"public"`
}

func PostGachas(c *gin.Context) {
	var gachaRequest model.GachaRequest
	c.Bind(&gachaRequest)
	request := mapRequest(gachaRequest)

	err := gacha.Validate(request)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	result, err := gacha.Execute(request)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	resultModel, err := mapResult(result, request, gachaRequest, c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := model.DB.Create(resultModel).Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	resultResponse, err := mapResultResponse(resultModel, c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, resultResponse)
}

type ResultResponseExt struct {
	ResultResponse
	NextAvailable bool `json:"nextAvailable"`
	NextID        uint `json:"nextId"`
	PrevAvailable bool `json:"prevAvailable"`
	PrevID        uint `json:"prevId"`
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

	resultResponse, err := mapResultResponse(&result, c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if result.UserID == userID {
		var nextResults []model.Result
		if err := model.DB.
			Where("time < ? AND game_title_id = ? AND user_id = ?", result.Time, result.GameTitleID, result.UserID).
			Order("time DESC").
			Limit(1).
			Find(&nextResults).
			Error; err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		nextAvailable := len(nextResults) > 0
		var nextID uint
		if len(nextResults) > 0 {
			nextID = nextResults[0].ID
		}

		var prevResults []model.Result
		if err := model.DB.
			Where("time > ? AND game_title_id = ? AND user_id = ?", result.Time, result.GameTitleID, result.UserID).
			Order("time ASC").
			Limit(1).
			Find(&prevResults).
			Error; err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		prevAvailable := len(prevResults) > 0
		var prevID uint
		if len(prevResults) > 0 {
			prevID = prevResults[0].ID
		}

		c.JSON(http.StatusOK, ResultResponseExt{
			ResultResponse: *resultResponse,
			NextAvailable:  nextAvailable,
			NextID:         nextID,
			PrevAvailable:  prevAvailable,
			PrevID:         prevID,
		})
		return
	} else {
		c.JSON(http.StatusOK, resultResponse)
		return
	}
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

func mapResult(
	result gacha.Result,
	request gacha.Request,
	gachaRequest model.GachaRequest,
	c *gin.Context,
) (*model.Result, error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	itemIDs := make([]uint, len(result.Items))
	for i, item := range result.Items {
		itemIDs[i] = item.ID
	}
	itemIDsJSON, err := json.Marshal(itemIDs)
	if err != nil {
		return nil, err
	}
	now := time.Now()

	preferred := getPreferredLanguage(c)
	var gameTitle model.GameTitle
	if err := model.DB.
		Preload("Translations").
		First(&gameTitle, gachaRequest.GameTitle.ID).
		Error; err != nil {
		return nil, err
	}
	i := getTranslationIndex(preferred, gameTitle)
	gameTitle.GameTitleTranslatable = gameTitle.Translations[i].GameTitleTranslatable
	accessToken, ok := c.Get("access_token")
	if !ok {
		return nil, err
	}
	userID := accessToken.(oauth2.TokenInfo).GetUserID()
	return &model.Result{
		Request:       datatypes.JSON(requestJSON),
		ItemIDs:       datatypes.JSON(itemIDsJSON),
		GoalsAchieved: result.GoalsAchieved,
		MoneySpent:    result.MoneySpent,
		Time:          now,
		GameTitle:     &gameTitle,
		UserID:        userID,
		Public:        false,
	}, nil
}

func mapResultResponse(
	result *model.Result,
	c *gin.Context,
) (*ResultResponse, error) {
	preferred := getPreferredLanguage(c)

	uniqueItemIDs, err := makeUniqueItemIDs(result)
	if err != nil {
		return nil, err
	}

	var items []model.Item
	if err := model.DB.
		Scopes(itemsByIDs(uniqueItemIDs)).
		Find(&items).
		Error; err != nil {
		return nil, err
	}
	for i := 0; i < len(items); i++ {
		j := getTranslationIndex(preferred, items[i])
		items[i].ItemTranslatable = items[i].Translations[j].ItemTranslatable
		k := getTranslationIndex(preferred, items[i].Tier)
		items[i].Tier.TierTranslatable = items[i].Tier.Translations[k].TierTranslatable
	}

	var request gacha.Request
	if err := json.Unmarshal(result.Request, &request); err != nil {
		return nil, err
	}

	remainingWantedItemIDs := make([]uint, 0)
	if request.Plan.ItemGoals {
		for wantedItemID := range request.Plan.WantedItems {
			found := false
			for _, item := range items {
				if item.ID == wantedItemID {
					found = true
					break
				}
			}
			if !found {
				remainingWantedItemIDs = append(remainingWantedItemIDs, wantedItemID)
			}
		}
	}
	var remainingWantedItems []model.Item
	if err := model.DB.
		Scopes(itemsByIDs(remainingWantedItemIDs)).
		Find(&remainingWantedItems).
		Error; err != nil {
		return nil, err
	}
	for i := 0; i < len(remainingWantedItems); i++ {
		j := getTranslationIndex(preferred, remainingWantedItems[i])
		remainingWantedItems[i].ItemTranslatable = remainingWantedItems[i].Translations[j].ItemTranslatable
		k := getTranslationIndex(preferred, remainingWantedItems[i].Tier)
		remainingWantedItems[i].Tier.TierTranslatable = remainingWantedItems[i].Tier.Translations[k].TierTranslatable
	}

	remainingWantedTierIDs := make([]uint, 0)
	if request.Plan.TierGoals {
		for wantedTierID := range request.Plan.WantedTiers {
			found := false
			for _, item := range items {
				if item.Tier.ID == wantedTierID {
					found = true
					break
				}
			}
			if !found {
				remainingWantedTierIDs = append(remainingWantedTierIDs, wantedTierID)
			}
		}
	}
	var remainingWantedTiers []model.Tier
	if err := model.DB.
		Scopes(tiersByIDs(remainingWantedTierIDs)).
		Find(&remainingWantedTiers).
		Error; err != nil {
		return nil, err
	}
	for i := 0; i < len(remainingWantedTiers); i++ {
		j := getTranslationIndex(preferred, remainingWantedTiers[i])
		remainingWantedTiers[i].TierTranslatable = remainingWantedTiers[i].Translations[j].TierTranslatable
	}

	return &ResultResponse{
		Result:               result,
		Items:                items,
		RemainingWantedItems: remainingWantedItems,
		RemainingWantedTiers: remainingWantedTiers,
	}, nil
}

func itemsByIDs(itemIDs []uint) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Model(&model.Item{}).
			Preload("Tier.Translations").
			Preload("Translations").
			Where("id IN ?", itemIDs)
	}
}

func tiersByIDs(tierIDs []uint) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Model(&model.Tier{}).
			Preload("Translations").
			Where("id IN ?", tierIDs)
	}
}

func makeUniqueItemIDs(result *model.Result) ([]uint, error) {
	uniqueItemIDs := make([]uint, 0)
	itemIDMap := make(map[uint]bool)
	var itemIDs []uint
	if err := json.Unmarshal(result.ItemIDs, &itemIDs); err != nil {
		return nil, err
	}
	for _, itemID := range itemIDs {
		itemIDMap[itemID] = true
	}
	for itemID := range itemIDMap {
		uniqueItemIDs = append(uniqueItemIDs, itemID)
	}
	return uniqueItemIDs, nil
}
