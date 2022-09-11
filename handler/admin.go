package handler

import (
	"gacha-simulator/model"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GameTitleWrapper struct {
	GameTitle model.GameTitle  `json:"gameTitle"`
	Tiers     []model.Tier     `json:"tiers"`
	Items     []model.Item     `json:"items"`
	Pricings  []model.Pricing  `json:"pricings"`
	Policies  []model.Policies `json:"policies"`
	Plans     []model.Plan     `json:"plans"`
}

type GameTitles struct {
	GameTitleWrappers []GameTitleWrapper `json:"gameTitles"`
}

type Result struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failure int `json:"failure"`
}

func PostGameTitlesBulk(ctx *gin.Context) {
	result := Result{}
	var gameTitles GameTitles
	err := ctx.Bind(&gameTitles)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	for _, gameTitleWrapper := range gameTitles.GameTitleWrappers {
		err := model.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&gameTitleWrapper.GameTitle).Error; err != nil {
				return err
			}
			if err := tx.Create(&gameTitleWrapper.Tiers).Error; err != nil {
				return err
			}
			if err := tx.Create(&gameTitleWrapper.Items).Error; err != nil {
				return err
			}
			if err := tx.Create(&gameTitleWrapper.Pricings).Error; err != nil {
				return err
			}
			if err := tx.Create(&gameTitleWrapper.Policies).Error; err != nil {
				return err
			}
			if err := tx.Create(&gameTitleWrapper.Plans).Error; err != nil {
				return err
			}
			return nil
		})
		result.Total++
		if err == nil {
			result.Success++
		} else {
			result.Failure++
		}
	}
	ctx.JSON(http.StatusOK, result)
}

func DeleteGameTitle(ctx *gin.Context) {
	gameTitleSlug := ctx.Param("gameTitleSlug")
	if err := model.DB.Where("slug = ?", gameTitleSlug).Delete(&model.GameTitle{}).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.Status(http.StatusNoContent)
}
