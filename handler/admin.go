package handler

import (
	"encoding/json"
	"errors"
	"gacha-simulator/model"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GameTitleInput struct {
	Slug         string                      `json:"slug"`
	ImageURL     string                      `json:"imageUrl"`
	DisplayOrder uint                        `json:"displayOrder"`
	Translations []GameTitleTranslationInput `json:"translations"`
}

type GameTitleTranslationInput struct {
	Language    string `json:"language"`
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Description string `json:"description"`
}

type TierInput struct {
	Key          string                 `json:"key"`
	Ratio        int                    `json:"ratio"`
	ImageURL     string                 `json:"imageUrl"`
	Translations []TierTranslationInput `json:"translations"`
}

type TierTranslationInput struct {
	Language  string `json:"language"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
}

type ItemInput struct {
	TierKey      string                 `json:"tierKey"`
	Key          *string                `json:"key"`
	Ratio        int                    `json:"ratio"`
	ImageURL     string                 `json:"imageUrl"`
	Translations []ItemTranslationInput `json:"translations"`
}

type ItemTranslationInput struct {
	Language  string `json:"language"`
	Name      string `json:"name" gorm:"index"`
	ShortName string `json:"shortName" gorm:"index"`
}

type PricingInput struct {
	Key                     *string                   `json:"key"`
	PricePerGacha           float64                   `json:"pricePerGacha"`
	Discount                bool                      `json:"discount"`
	DiscountTrigger         int                       `json:"discountTrigger"`
	DiscountedPricePerGacha float64                   `json:"discountedPricePerGacha"`
	Translations            []PricingTranslationInput `json:"translations"`
}

type PricingTranslationInput struct {
	Language string `json:"language"`
	Name     string `json:"name"`
}

type PoliciesInput struct {
	Key          *string                    `json:"key"`
	Pity         bool                       `json:"pity"`
	PityTrigger  int                        `json:"pityTrigger"`
	PityItemKey  *string                    `json:"pityItemKey"`
	Translations []PoliciesTranslationInput `json:"translations"`
}

type PoliciesTranslationInput struct {
	Language string `json:"language"`
	Name     string `json:"name"`
}

type KeyNumberTuple struct {
	Key    string `json:"key"`
	Number int    `json:"number"`
}

type PlanInput struct {
	Key                  *string                `json:"key"`
	Budget               float64                `json:"budget"`
	MaxConsecutiveGachas int                    `json:"maxConsecutiveGachas"`
	ItemGoals            bool                   `json:"itemGoals"`
	WantedItems          []KeyNumberTuple       `json:"wantedItems"`
	TierGoals            bool                   `json:"tierGoals"`
	WantedTiers          []KeyNumberTuple       `json:"wantedTiers"`
	Translations         []PlanTranslationInput `json:"translations"`
}

type PlanTranslationInput struct {
	Language string `json:"language"`
	Name     string `json:"name"`
}

type PresetInput struct {
	PricingKey   *string                  `json:"pricingKey"`
	PoliciesKey  *string                  `json:"policiesKey"`
	PlanKey      *string                  `json:"planKey"`
	Translations []PresetTranslationInput `json:"translations"`
}

type PresetTranslationInput struct {
	Language    string `json:"language"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type GameTitleBulk struct {
	GameTitle GameTitleInput  `json:"gameTitle"`
	Tiers     []TierInput     `json:"tiers"`
	Items     []ItemInput     `json:"items"`
	Pricings  []PricingInput  `json:"pricings"`
	Policies  []PoliciesInput `json:"policies"`
	Plans     []PlanInput     `json:"plans"`
	Presets   []PresetInput   `json:"presets"`
}

type GameTitleBulkRequest struct {
	GameTitleBulks []GameTitleBulk `json:"gameTitleBulks"`
}

type IndexErrorTuple struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

type GameTitleBulkResponse struct {
	Success int               `json:"success"`
	Failure int               `json:"failure"`
	Errors  []IndexErrorTuple `json:"errors"`
}

func PostGameTitlesBulk(ctx *gin.Context) {
	var gameTitleBulkRequest GameTitleBulkRequest
	err := ctx.Bind(&gameTitleBulkRequest)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	response := GameTitleBulkResponse{
		Success: 0,
		Failure: 0,
		Errors:  make([]IndexErrorTuple, 0),
	}
	for i, gameTitleBulk := range gameTitleBulkRequest.GameTitleBulks {
		if err := model.DB.Transaction(func(tx *gorm.DB) error {
			gameTitleModel := mapGameTitleModel(gameTitleBulk.GameTitle)
			if err := tx.Create(gameTitleModel).Error; err != nil {
				return err
			}
			gameTitleID := gameTitleModel.ID
			tierKeyToModel := make(map[string]*model.Tier)
			tiersModel := mapTiersModel(gameTitleBulk.Tiers, gameTitleID, tierKeyToModel)
			if err := tx.Create(tiersModel).Error; err != nil {
				return err
			}
			itemKeyToModel := make(map[string]*model.Item)
			itemsModel, err := mapItemsModel(gameTitleBulk.Items, tierKeyToModel, itemKeyToModel)
			if err != nil {
				return err
			}
			if err := tx.Create(itemsModel).Error; err != nil {
				return err
			}
			pricingKeyToModel := make(map[string]*model.Pricing)
			pricingsModel := mapPricingsModel(gameTitleBulk.Pricings, gameTitleID, pricingKeyToModel)
			if err := tx.Create(pricingsModel).Error; err != nil {
				return err
			}
			policiesKeyToModel := make(map[string]*model.Policies)
			policiesModel, err := mapPoliciesModel(gameTitleBulk.Policies, gameTitleID, itemKeyToModel, policiesKeyToModel)
			if err != nil {
				return err
			}
			if err := tx.Create(policiesModel).Error; err != nil {
				return err
			}
			planKeyToModel := make(map[string]*model.Plan)
			plansModel, err := mapPlansModel(gameTitleBulk.Plans, gameTitleID, tierKeyToModel, itemKeyToModel, planKeyToModel)
			if err != nil {
				return err
			}
			if err := tx.Create(plansModel).Error; err != nil {
				return err
			}
			presetsModel, err := mapPresetsModel(gameTitleBulk.Presets, gameTitleID, pricingKeyToModel, policiesKeyToModel, planKeyToModel)
			if err != nil {
				return err
			}
			if err := tx.Create(presetsModel).Error; err != nil {
				return err
			}
			return nil
		}); err != nil {
			response.Failure++
			response.Errors = append(response.Errors, IndexErrorTuple{
				Index: i,
				Error: err.Error(),
			})
		} else {
			response.Success++
		}
	}
	ctx.JSON(http.StatusOK, &response)
}

func DeleteGameTitle(ctx *gin.Context) {
	gameTitleSlug := ctx.Param("gameTitleSlug")
	if err := model.DB.Where("slug = ?", gameTitleSlug).Delete(&model.GameTitle{}).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func mapGameTitleModel(gameTitleInput GameTitleInput) *model.GameTitle {
	translations := mapGameTitleTranslationsModel(gameTitleInput.Translations)
	return &model.GameTitle{
		Slug:         gameTitleInput.Slug,
		ImageURL:     gameTitleInput.ImageURL,
		DisplayOrder: gameTitleInput.DisplayOrder,
		Translations: translations,
	}
}

func mapTiersModel(
	tiersInput []TierInput,
	gameTitleID uint,
	tierKeyToModel map[string]*model.Tier,
) []*model.Tier {
	tiersModel := make([]*model.Tier, 0)
	for i := 0; i < len(tiersInput); i++ {
		tierModel := mapTierModel(tiersInput[i], gameTitleID, tierKeyToModel)
		tiersModel = append(tiersModel, tierModel)
	}
	return tiersModel
}

func mapTierModel(
	tierInput TierInput,
	gameTitleID uint,
	tierKeyToModel map[string]*model.Tier,
) *model.Tier {
	translations := mapTierTranslationsModel(tierInput.Translations)
	tierModel := model.Tier{
		Ratio:        tierInput.Ratio,
		GameTitleID:  gameTitleID,
		ImageURL:     tierInput.ImageURL,
		Translations: translations,
	}
	tierKeyToModel[tierInput.Key] = &tierModel
	return &tierModel
}

func mapItemsModel(
	itemsInput []ItemInput,
	tierKeyToModel map[string]*model.Tier,
	itemKeyToModel map[string]*model.Item,
) ([]*model.Item, error) {
	itemsModel := make([]*model.Item, 0)
	for i := 0; i < len(itemsInput); i++ {
		itemModel, err := mapItemModel(itemsInput[i], tierKeyToModel, itemKeyToModel)
		if err != nil {
			return nil, err
		}
		itemsModel = append(itemsModel, itemModel)
	}
	return itemsModel, nil
}

func mapItemModel(
	itemInput ItemInput,
	tierKeyToModel map[string]*model.Tier,
	itemKeyToModel map[string]*model.Item,
) (*model.Item, error) {
	translations := mapItemTranslationsModel(itemInput.Translations)
	itemModel := model.Item{
		Ratio:        itemInput.Ratio,
		ImageURL:     itemInput.ImageURL,
		Translations: translations,
	}
	if tier, ok := tierKeyToModel[itemInput.TierKey]; ok {
		itemModel.TierID = tier.ID
	} else {
		return nil, errors.New("invalid TierKey: " + itemInput.TierKey)
	}
	if itemInput.Key != nil && *itemInput.Key != "" {
		itemKeyToModel[*itemInput.Key] = &itemModel
	}
	return &itemModel, nil
}

func mapPricingsModel(
	pricingsInput []PricingInput,
	gameTitleID uint,
	pricingKeyToModel map[string]*model.Pricing,
) []*model.Pricing {
	pricingsModel := make([]*model.Pricing, 0)
	for i := 0; i < len(pricingsInput); i++ {
		pricingModel := mapPricingModel(pricingsInput[i], gameTitleID, pricingKeyToModel)
		pricingsModel = append(pricingsModel, pricingModel)
	}
	return pricingsModel
}

func mapPricingModel(
	pricingInput PricingInput,
	gameTitleID uint,
	pricingKeyToModel map[string]*model.Pricing,
) *model.Pricing {
	translations := mapPricingTranslationsModel(pricingInput.Translations)
	pricingModel := model.Pricing{
		PricePerGacha:           pricingInput.PricePerGacha,
		Discount:                pricingInput.Discount,
		DiscountTrigger:         pricingInput.DiscountTrigger,
		DiscountedPricePerGacha: pricingInput.DiscountedPricePerGacha,
		GameTitleID:             gameTitleID,
		Translations:            translations,
	}
	if pricingInput.Key != nil && *pricingInput.Key != "" {
		pricingKeyToModel[*pricingInput.Key] = &pricingModel
	}
	return &pricingModel
}

func mapPoliciesModel(
	policiesInput []PoliciesInput,
	gameTitleID uint,
	itemKeyToModel map[string]*model.Item,
	policiesKeyToModel map[string]*model.Policies,
) ([]*model.Policies, error) {
	policiesModel := make([]*model.Policies, 0)
	for i := 0; i < len(policiesInput); i++ {
		policyModel, err := mapPolicyModel(policiesInput[i], gameTitleID, itemKeyToModel, policiesKeyToModel)
		if err != nil {
			return nil, err
		}
		policiesModel = append(policiesModel, policyModel)
	}
	return policiesModel, nil
}

func mapPolicyModel(
	policiesInput PoliciesInput,
	gameTitleID uint,
	itemKeyToModel map[string]*model.Item,
	policiesKeyToModel map[string]*model.Policies,
) (*model.Policies, error) {
	translations := mapPoliciesTranslationsModel(policiesInput.Translations)
	policiesModel := model.Policies{
		Pity:         policiesInput.Pity,
		PityTrigger:  policiesInput.PityTrigger,
		GameTitleID:  gameTitleID,
		Translations: translations,
	}
	if policiesInput.Pity && policiesInput.PityItemKey != nil && *policiesInput.PityItemKey != "" {
		if pityItem, ok := itemKeyToModel[*policiesInput.PityItemKey]; ok {
			policiesModel.PityItemID = &pityItem.ID
		} else {
			return nil, errors.New("invalid PityItemKey: " + *policiesInput.PityItemKey)
		}
	}
	if policiesInput.Key != nil && *policiesInput.Key != "" {
		policiesKeyToModel[*policiesInput.Key] = &policiesModel
	}
	return &policiesModel, nil
}

func mapPlansModel(
	plansInput []PlanInput,
	gameTitleID uint,
	tierKeyToModel map[string]*model.Tier,
	itemKeyToModel map[string]*model.Item,
	planKeyToModel map[string]*model.Plan,
) ([]*model.Plan, error) {
	plansModel := make([]*model.Plan, 0)
	for i := 0; i < len(plansInput); i++ {
		planModel, err := mapPlanModel(plansInput[i], gameTitleID, tierKeyToModel, itemKeyToModel, planKeyToModel)
		if err != nil {
			return nil, err
		}
		plansModel = append(plansModel, planModel)
	}
	return plansModel, nil
}

func mapPlanModel(
	planInput PlanInput,
	gameTitleID uint,
	tierKeyToModel map[string]*model.Tier,
	itemKeyToModel map[string]*model.Item,
	planKeyToModel map[string]*model.Plan,
) (*model.Plan, error) {
	translations := mapPlanTranslationsModel(planInput.Translations)
	planModel := model.Plan{
		Budget:               planInput.Budget,
		MaxConsecutiveGachas: planInput.MaxConsecutiveGachas,
		ItemGoals:            planInput.ItemGoals,
		WantedItemsJSON:      nil,
		TierGoals:            planInput.TierGoals,
		WantedTiersJSON:      nil,
		GameTitleID:          gameTitleID,
		Translations:         translations,
	}
	if planInput.ItemGoals && len(planInput.WantedItems) > 0 {
		wantedItemsMap := make(map[uint]int, 0)
		for _, wantedItem := range planInput.WantedItems {
			if wantedItemModel, ok := itemKeyToModel[wantedItem.Key]; ok {
				itemID := wantedItemModel.ID
				wantedItemsMap[itemID] = wantedItem.Number
			} else {
				return nil, errors.New("invalid WantedItem Key: " + wantedItem.Key)
			}
		}
		wantedItemsJSON, err := json.Marshal(wantedItemsMap)
		if err != nil {
			return nil, err
		}
		planModel.WantedItemsJSON = wantedItemsJSON
	}
	if planInput.TierGoals && len(planInput.WantedTiers) > 0 {
		wantedTiersMap := make(map[uint]int, 0)
		for _, wantedTier := range planInput.WantedTiers {
			if wantedTierModel, ok := tierKeyToModel[wantedTier.Key]; ok {
				tierID := wantedTierModel.ID
				wantedTiersMap[tierID] = wantedTier.Number
			} else {
				return nil, errors.New("invalid WantedTier Key: " + wantedTier.Key)
			}
		}
		wantedTiersJSON, err := json.Marshal(wantedTiersMap)
		if err != nil {
			return nil, err
		}
		planModel.WantedTiersJSON = wantedTiersJSON
	}
	if planInput.Key != nil && *planInput.Key != "" {
		planKeyToModel[*planInput.Key] = &planModel
	}
	return &planModel, nil
}

func mapPresetsModel(
	presetsInput []PresetInput,
	gameTitleID uint,
	pricingKeyToModel map[string]*model.Pricing,
	policiesKeyToModel map[string]*model.Policies,
	planKeyToModel map[string]*model.Plan,
) ([]*model.Preset, error) {
	presetsModel := make([]*model.Preset, 0)
	for i := 0; i < len(presetsInput); i++ {
		presetModel, err := mapPresetModel(presetsInput[i], gameTitleID, pricingKeyToModel, policiesKeyToModel, planKeyToModel)
		if err != nil {
			return nil, err
		}
		presetsModel = append(presetsModel, presetModel)
	}
	return presetsModel, nil
}

func mapPresetModel(
	presetInput PresetInput,
	gameTitleID uint,
	pricingKeyToModel map[string]*model.Pricing,
	policiesKeyToModel map[string]*model.Policies,
	planKeyToModel map[string]*model.Plan,
) (*model.Preset, error) {
	translations := mapPresetTranslationsModel(presetInput.Translations)
	presetModel := model.Preset{
		GameTitleID:  gameTitleID,
		Translations: translations,
	}
	if presetInput.PricingKey != nil && *presetInput.PricingKey != "" {
		if pricingModel, ok := pricingKeyToModel[*presetInput.PricingKey]; ok {
			presetModel.PricingID = &pricingModel.ID
		} else {
			return nil, errors.New("invalid PricingKey: " + *presetInput.PricingKey)
		}
	}
	if presetInput.PoliciesKey != nil && *presetInput.PoliciesKey != "" {
		if policiesModel, ok := policiesKeyToModel[*presetInput.PoliciesKey]; ok {
			presetModel.PoliciesID = &policiesModel.ID
		} else {
			return nil, errors.New("invalid PoliciesKey: " + *presetInput.PoliciesKey)
		}
	}
	if presetInput.PlanKey != nil && *presetInput.PlanKey != "" {
		if planModel, ok := planKeyToModel[*presetInput.PlanKey]; ok {
			presetModel.PlanID = &planModel.ID
		} else {
			return nil, errors.New("invalid PlanKey: " + *presetInput.PlanKey)
		}
	}
	return &presetModel, nil
}

func mapGameTitleTranslationsModel(translationsInput []GameTitleTranslationInput) []model.GameTitleTranslation {
	translations := make([]model.GameTitleTranslation, 0)
	for i := 0; i < len(translationsInput); i++ {
		translation := mapGameTitleTranslationModel(translationsInput[i])
		translations = append(translations, *translation)
	}
	return translations
}

func mapGameTitleTranslationModel(translationInput GameTitleTranslationInput) *model.GameTitleTranslation {
	return &model.GameTitleTranslation{
		Language: translationInput.Language,
		GameTitleTranslatable: model.GameTitleTranslatable{
			Name:        translationInput.Name,
			ShortName:   translationInput.ShortName,
			Description: translationInput.Description,
		},
	}
}

func mapTierTranslationsModel(translationsInput []TierTranslationInput) []model.TierTranslation {
	translations := make([]model.TierTranslation, 0)
	for i := 0; i < len(translationsInput); i++ {
		translation := mapTierTranslationModel(translationsInput[i])
		translations = append(translations, *translation)
	}
	return translations
}

func mapTierTranslationModel(translationInput TierTranslationInput) *model.TierTranslation {
	return &model.TierTranslation{
		Language: translationInput.Language,
		TierTranslatable: model.TierTranslatable{
			Name:      translationInput.Name,
			ShortName: translationInput.ShortName,
		},
	}
}

func mapItemTranslationsModel(translationsInput []ItemTranslationInput) []model.ItemTranslation {
	translations := make([]model.ItemTranslation, 0)
	for i := 0; i < len(translationsInput); i++ {
		translation := mapItemTranslationModel(translationsInput[i])
		translations = append(translations, *translation)
	}
	return translations
}

func mapItemTranslationModel(translationInput ItemTranslationInput) *model.ItemTranslation {
	return &model.ItemTranslation{
		Language: translationInput.Language,
		ItemTranslatable: model.ItemTranslatable{
			Name:      translationInput.Name,
			ShortName: translationInput.ShortName,
		},
	}
}

func mapPricingTranslationsModel(translationsInput []PricingTranslationInput) []model.PricingTranslation {
	translations := make([]model.PricingTranslation, 0)
	for i := 0; i < len(translationsInput); i++ {
		translation := mapPricingTranslationModel(translationsInput[i])
		translations = append(translations, *translation)
	}
	return translations
}

func mapPricingTranslationModel(translationInput PricingTranslationInput) *model.PricingTranslation {
	return &model.PricingTranslation{
		Language: translationInput.Language,
		PricingTranslatable: model.PricingTranslatable{
			Name: translationInput.Name,
		},
	}
}

func mapPoliciesTranslationsModel(translationsInput []PoliciesTranslationInput) []model.PoliciesTranslation {
	translations := make([]model.PoliciesTranslation, 0)
	for i := 0; i < len(translationsInput); i++ {
		translation := mapPoliciesTranslationModel(translationsInput[i])
		translations = append(translations, *translation)
	}
	return translations
}

func mapPoliciesTranslationModel(translationInput PoliciesTranslationInput) *model.PoliciesTranslation {
	return &model.PoliciesTranslation{
		Language: translationInput.Language,
		PoliciesTranslatable: model.PoliciesTranslatable{
			Name: translationInput.Name,
		},
	}
}

func mapPlanTranslationsModel(translationsInput []PlanTranslationInput) []model.PlanTranslation {
	translations := make([]model.PlanTranslation, 0)
	for i := 0; i < len(translationsInput); i++ {
		translation := mapPlanTranslationModel(translationsInput[i])
		translations = append(translations, *translation)
	}
	return translations
}

func mapPlanTranslationModel(translationInput PlanTranslationInput) *model.PlanTranslation {
	return &model.PlanTranslation{
		Language: translationInput.Language,
		PlanTranslatable: model.PlanTranslatable{
			Name: translationInput.Name,
		},
	}
}

func mapPresetTranslationsModel(translationsInput []PresetTranslationInput) []model.PresetTranslation {
	translations := make([]model.PresetTranslation, 0)
	for i := 0; i < len(translationsInput); i++ {
		translation := mapPresetTranslationModel(translationsInput[i])
		translations = append(translations, *translation)
	}
	return translations
}

func mapPresetTranslationModel(translationInput PresetTranslationInput) *model.PresetTranslation {
	return &model.PresetTranslation{
		Language: translationInput.Language,
		PresetTranslatable: model.PresetTranslatable{
			Name:        translationInput.Name,
			Description: translationInput.Description,
		},
	}
}
