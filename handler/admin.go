package handler

import (
	"gacha-simulator/model"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GameTitleBulk struct {
	GameTitle model.GameTitle  `json:"gameTitle"`
	Tiers     []model.Tier     `json:"tiers"`
	Items     []model.Item     `json:"items"`
	Pricings  []model.Pricing  `json:"pricings"`
	Policies  []model.Policies `json:"policies"`
	Plans     []model.Plan     `json:"plans"`
	Presets   []model.Preset   `json:"presets"`
}

type GameTitleBulkRequest struct {
	GameTitleBulks []GameTitleBulk `json:"gameTitleBulks"`
}

type GameTitleBulkResponse struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failure int `json:"failure"`
}

func PostGameTitlesBulk(ctx *gin.Context) {
	response := GameTitleBulkResponse{}
	var gameTitleBulkRequest GameTitleBulkRequest
	err := ctx.Bind(&gameTitleBulkRequest)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	for _, gameTitleBulk := range gameTitleBulkRequest.GameTitleBulks {
		err := model.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&gameTitleBulk.GameTitle).Error; err != nil {
				return err
			}
			if err := tx.Create(&gameTitleBulk.Tiers).Error; err != nil {
				return err
			}
			if err := tx.Create(&gameTitleBulk.Items).Error; err != nil {
				return err
			}
			if err := tx.Create(&gameTitleBulk.Pricings).Error; err != nil {
				return err
			}
			if err := tx.Create(&gameTitleBulk.Policies).Error; err != nil {
				return err
			}
			if err := tx.Create(&gameTitleBulk.Plans).Error; err != nil {
				return err
			}
			if err := tx.Create(&gameTitleBulk.Presets).Error; err != nil {
				return err
			}
			return nil
		})
		response.Total++
		if err == nil {
			response.Success++
		} else {
			response.Failure++
		}
	}
	ctx.JSON(http.StatusOK, response)
}

func DeleteGameTitle(ctx *gin.Context) {
	gameTitleSlug := ctx.Param("gameTitleSlug")
	if err := model.DB.Where("slug = ?", gameTitleSlug).Delete(&model.GameTitle{}).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.Status(http.StatusNoContent)
}
