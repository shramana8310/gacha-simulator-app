package handler

import (
	"encoding/json"
	"errors"
	"gacha-simulator/gacha"
	"gacha-simulator/model"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type GachaRequest struct {
	GameTitle     GameTitle `json:"gameTitle"`
	Tiers         []Tier    `json:"tiers"`
	ItemsIncluded bool      `json:"itemsIncluded"`
	Pricing       Pricing   `json:"pricing"`
	Policies      Policies  `json:"policies"`
	Plan          Plan      `json:"plan"`
}

type ResultResponse struct {
	Result
	Items                []Item `json:"items"`
	RemainingWantedItems []Item `json:"remainingWantedItems"`
	RemainingWantedTiers []Tier `json:"remainingWantedTiers"`
}

type ResultResponseExt struct {
	ResultResponse
	NextAvailable bool `json:"nextAvailable"`
	NextID        uint `json:"nextId"`
	PrevAvailable bool `json:"prevAvailable"`
	PrevID        uint `json:"prevId"`
}

type PatchGachaRequest struct {
	Public bool `json:"public"`
}

func PostGachas(c *gin.Context) {
	var gachaRequest GachaRequest
	c.Bind(&gachaRequest)
	request := mapGachaRequest(gachaRequest)

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

	resultModel, err := mapResultModel(result, request, gachaRequest, c)
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

func GetGacha(c *gin.Context) {
	resultID := c.Param("resultID")
	resultModel, err := getResultModel(resultID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if resultModel == nil {
		c.Status(http.StatusNotFound)
		return
	}
	userID, ok := getUserID(c)
	if !ok {
		c.Status(http.StatusBadRequest)
		return
	}
	if !resultModel.Public && resultModel.UserID != userID {
		c.Status(http.StatusForbidden)
		return
	}
	resultResponse, err := mapResultResponse(resultModel, c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if resultModel.UserID == userID {
		nextResultsModel, err := getNextResultsModel(*resultModel)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		nextAvailable := len(nextResultsModel) > 0
		var nextID uint
		if len(nextResultsModel) > 0 {
			nextID = nextResultsModel[0].ID
		}
		prevResultsModel, err := getPrevResultsModel(*resultModel)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		prevAvailable := len(prevResultsModel) > 0
		var prevID uint
		if len(prevResultsModel) > 0 {
			prevID = prevResultsModel[0].ID
		}
		c.JSON(http.StatusOK, &ResultResponseExt{
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
	userID, ok := getUserID(c)
	if !ok {
		c.Status(http.StatusBadRequest)
		return
	}

	resultModel, err := getResultModelByIDAndUserID(resultID, userID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if resultModel == nil {
		c.Status(http.StatusNotFound)
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
	userID, ok := getUserID(c)
	if !ok {
		c.Status(http.StatusBadRequest)
		return
	}

	resultModel, err := getResultModelByIDAndUserID(resultID, userID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if resultModel == nil {
		c.Status(http.StatusNotFound)
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

func getItemsModelByIDs(itemIDs []uint) ([]model.Item, error) {
	var itemsModel []model.Item
	if err := model.DB.
		Scopes(itemsByIDs(itemIDs)).
		Find(&itemsModel).
		Error; err != nil {
		return nil, err
	}
	return itemsModel, nil
}

func getTiersModelByIDs(tierIDs []uint) ([]model.Tier, error) {
	var tiersModel []model.Tier
	if err := model.DB.
		Scopes(tiersByIDs(tierIDs)).
		Find(&tiersModel).
		Error; err != nil {
		return nil, err
	}
	return tiersModel, nil
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

func getResultModel(resultID string) (*model.Result, error) {
	var resultModel model.Result
	if err := model.DB.
		Preload("GameTitle.Translations").
		First(&resultModel, resultID).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &resultModel, nil
}

func getNextResultsModel(resultModel model.Result) ([]model.Result, error) {
	var nextResultsModel []model.Result
	if err := model.DB.
		Where("time < ? AND game_title_id = ? AND user_id = ?", resultModel.Time, resultModel.GameTitleID, resultModel.UserID).
		Order("time DESC").
		Limit(1).
		Find(&nextResultsModel).
		Error; err != nil {
		return nil, err
	}
	return nextResultsModel, nil
}

func getPrevResultsModel(resultModel model.Result) ([]model.Result, error) {
	var prevResultsModel []model.Result
	if err := model.DB.
		Where("time > ? AND game_title_id = ? AND user_id = ?", resultModel.Time, resultModel.GameTitleID, resultModel.UserID).
		Order("time ASC").
		Limit(1).
		Find(&prevResultsModel).
		Error; err != nil {
		return nil, err
	}
	return prevResultsModel, nil
}

func getResultModelByIDAndUserID(resultID, userID string) (*model.Result, error) {
	var resultModel model.Result
	if err := model.DB.
		Model(&model.Result{}).
		Where("id = ? AND user_id = ?", resultID, userID).
		First(&resultModel).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &resultModel, nil
}

func mapGachaRequest(gachaRequest GachaRequest) gacha.Request {
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
		GetItemFromIndex: func(tierID uint, index int) (*gacha.Item, error) {
			var item model.Item
			if err := model.DB.
				Model(&model.Item{}).
				Where("tier_id", tierID).
				Offset(index).
				Preload("Tier").
				First(&item).
				Error; err != nil {
				return nil, err
			}
			return &gacha.Item{
				ID:   item.ID,
				Tier: &gacha.Tier{ID: item.Tier.ID},
			}, nil
		},
		GetItemFromID: func(itemID uint) (*gacha.Item, error) {
			var item model.Item
			if err := model.DB.
				Preload("Tier").
				First(&item, "items.id=?", itemID).
				Error; err != nil {
				return nil, err
			}
			return &gacha.Item{
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

func mapResultModel(
	result gacha.Result,
	request gacha.Request,
	gachaRequest GachaRequest,
	c *gin.Context,
) (*model.Result, error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	itemIDs := make([]uint, 0)
	for _, item := range result.Items {
		itemIDs = append(itemIDs, item.ID)
	}
	itemIDsJSON, err := json.Marshal(itemIDs)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	userID, ok := getUserID(c)
	if !ok {
		return nil, err
	}
	return &model.Result{
		Request:       datatypes.JSON(requestJSON),
		ItemIDs:       datatypes.JSON(itemIDsJSON),
		GoalsAchieved: result.GoalsAchieved,
		MoneySpent:    result.MoneySpent,
		Time:          now,
		GameTitleID:   gachaRequest.GameTitle.ID,
		UserID:        userID,
		Public:        false,
	}, nil
}

func mapResultResponse(
	resultModel *model.Result,
	c *gin.Context,
) (*ResultResponse, error) {
	var request gacha.Request
	if err := json.Unmarshal(resultModel.Request, &request); err != nil {
		return nil, err
	}

	uniqueItemIDs, err := makeUniqueItemIDs(resultModel)
	if err != nil {
		return nil, err
	}
	itemsModel, err := getItemsModelByIDs(uniqueItemIDs)
	if err != nil {
		return nil, err
	}
	items := mapItems(itemsModel, c)

	remainingWantedItemIDs := makeRemainingWantedItemIDs(request, items)
	remainingWantedItemsModel, err := getItemsModelByIDs(remainingWantedItemIDs)
	if err != nil {
		return nil, err
	}
	remainingWantedItems := mapItems(remainingWantedItemsModel, c)

	remainingWantedTierIDs := makeRemainingWantedTierIDs(request, items)
	remainingWantedTiersModel, err := getTiersModelByIDs(remainingWantedTierIDs)
	if err != nil {
		return nil, err
	}
	remainingWantedTiers := mapTiers(remainingWantedTiersModel, c)

	result, err := mapResult(*resultModel, c)
	if err != nil {
		return nil, err
	}

	return &ResultResponse{
		Result:               *result,
		Items:                items,
		RemainingWantedItems: remainingWantedItems,
		RemainingWantedTiers: remainingWantedTiers,
	}, nil
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

func makeRemainingWantedItemIDs(request gacha.Request, items []Item) []uint {
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
	return remainingWantedItemIDs
}

func makeRemainingWantedTierIDs(request gacha.Request, items []Item) []uint {
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
	return remainingWantedTierIDs
}
