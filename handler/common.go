package handler

import (
	"gacha-simulator/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

type GameTitle struct {
	ID           uint   `json:"id"`
	Slug         string `json:"slug"`
	ImageURL     string `json:"imageUrl"`
	DisplayOrder uint   `json:"displayOrder"`
	Name         string `json:"name"`
	ShortName    string `json:"shortName"`
	Description  string `json:"description"`
}

type Tier struct {
	ID        uint   `json:"id"`
	Ratio     int    `json:"ratio"`
	ImageURL  string `json:"imageUrl"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
}

type TierWithNumber struct {
	Tier
	Number uint `json:"number"`
}

type Item struct {
	ID        uint   `json:"id"`
	Ratio     int    `json:"ratio"`
	ImageURL  string `json:"imageUrl"`
	Tier      *Tier  `json:"tier"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
}

type ItemWithNumber struct {
	Item
	Number uint `json:"number"`
}

type Pricing struct {
	ID                      uint    `json:"id"`
	PricePerGacha           float64 `json:"pricePerGacha"`
	Discount                bool    `json:"discount"`
	DiscountTrigger         int     `json:"discountTrigger"`
	DiscountedPricePerGacha float64 `json:"discountedPricePerGacha"`
	Name                    string  `json:"name"`
}

type Policies struct {
	ID          uint   `json:"id"`
	Pity        bool   `json:"pity"`
	PityTrigger int    `json:"pityTrigger"`
	PityItem    *Item  `json:"pityItem"`
	Name        string `json:"name"`
}

type Plan struct {
	ID                   uint             `json:"id"`
	Budget               float64          `json:"budget"`
	MaxConsecutiveGachas int              `json:"maxConsecutiveGachas"`
	ItemGoals            bool             `json:"itemGoals"`
	WantedItems          []ItemWithNumber `json:"wantedItems"`
	TierGoals            bool             `json:"tierGoals"`
	WantedTiers          []TierWithNumber `json:"wantedTiers"`
	Name                 string           `json:"name"`
}

type Preset struct {
	ID          uint      `json:"id"`
	Pricing     *Pricing  `json:"pricing"`
	Policies    *Policies `json:"policies"`
	Plan        *Plan     `json:"plan"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

func getTranslationIndex(preferred []language.Tag, translationHolder model.TranslationHolder) int {
	var tags []language.Tag
	languageHolders := translationHolder.GetLanguageHolders()
	for i := 0; i < len(languageHolders); i++ {
		tag := language.Make(languageHolders[i].GetLanguage())
		tags = append(tags, tag)
	}
	matcher := language.NewMatcher(tags)
	_, i, _ := matcher.Match(preferred...)
	return i
}

func getPreferredLanguage(c *gin.Context) []language.Tag {
	preferred, _, err := language.ParseAcceptLanguage(c.GetHeader("Accept-Language"))
	if err != nil {
		return []language.Tag{language.English}
	}
	return preferred
}
