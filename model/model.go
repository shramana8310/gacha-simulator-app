package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

type TranslationHolder interface {
	GetLanguageHolders() []LanguageHolder
}

type LanguageHolder struct {
	GetLanguage func() string
}

type GameTitle struct {
	ID                    uint                   `json:"id"`
	Slug                  string                 `json:"slug" gorm:"size:256;unique;index;notNull"`
	ImageURL              string                 `json:"imageUrl"`
	Translations          []GameTitleTranslation `json:"translations" gorm:"constraint:OnDelete:CASCADE;"`
	GameTitleTranslatable `gorm:"-"`
}

type GameTitleTranslatable struct {
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Description string `json:"description"`
}

type GameTitleTranslation struct {
	ID          uint       `json:"id"`
	Language    string     `json:"language"`
	GameTitle   *GameTitle `json:"-"`
	GameTitleID uint       `json:"-"`
	GameTitleTranslatable
}

func (gameTitle GameTitle) GetLanguageHolders() []LanguageHolder {
	var languageHolders []LanguageHolder
	for i := 0; i < len(gameTitle.Translations); i++ {
		languageHolders = append(languageHolders, LanguageHolder{
			GetLanguage: func(i int) func() string {
				return func() string {
					return gameTitle.Translations[i].Language
				}
			}(i),
		})
	}
	return languageHolders
}

type Item struct {
	ID               uint              `json:"id"`
	Ratio            int               `json:"ratio"`
	ImageURL         string            `json:"imageUrl"`
	Tier             *Tier             `json:"tier,omitempty"`
	TierID           uint              `json:"tierId,omitempty"`
	Translations     []ItemTranslation `json:"translations" gorm:"constraint:OnDelete:CASCADE;"`
	ItemTranslatable `gorm:"-"`
}

type ItemTranslatable struct {
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
}

type ItemTranslation struct {
	ID       uint   `json:"id"`
	Language string `json:"language"`
	Item     *Item  `json:"-"`
	ItemID   uint   `json:"-"`
	ItemTranslatable
}

type ItemWithNumber struct {
	Item
	Number uint `json:"number"`
}

func (item Item) GetLanguageHolders() []LanguageHolder {
	var languageHolders []LanguageHolder
	for i := 0; i < len(item.Translations); i++ {
		languageHolders = append(languageHolders, LanguageHolder{
			GetLanguage: func(i int) func() string {
				return func() string {
					return item.Translations[i].Language
				}
			}(i),
		})
	}
	return languageHolders
}

type Tier struct {
	ID               uint              `json:"id"`
	Ratio            int               `json:"ratio"`
	Items            []Item            `json:"items,omitempty" gorm:"constraint:OnDelete:CASCADE;"`
	GameTitle        *GameTitle        `json:"gameTitle,omitempty" gorm:"constraint:OnDelete:CASCADE;"`
	GameTitleID      uint              `json:"gameTitleId,omitempty"`
	ImageURL         string            `json:"imageUrl"`
	Translations     []TierTranslation `json:"translations" gorm:"constraint:OnDelete:CASCADE;"`
	TierTranslatable `gorm:"-"`
}

type TierTranslatable struct {
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
}

type TierTranslation struct {
	ID       uint   `json:"id"`
	Language string `json:"language"`
	Tier     *Tier  `json:"-"`
	TierID   uint   `json:"-"`
	TierTranslatable
}

type TierWithNumber struct {
	Tier
	Number uint `json:"number"`
}

func (tier Tier) GetLanguageHolders() []LanguageHolder {
	var languageHolders []LanguageHolder
	for i := 0; i < len(tier.Translations); i++ {
		languageHolders = append(languageHolders, LanguageHolder{
			GetLanguage: func(i int) func() string {
				return func() string {
					return tier.Translations[i].Language
				}
			}(i),
		})
	}
	return languageHolders
}

type Pricing struct {
	ID                      uint                 `json:"id"`
	PricePerGacha           float64              `json:"pricePerGacha"`
	Discount                bool                 `json:"discount"`
	DiscountTrigger         int                  `json:"discountTrigger"`
	DiscountedPricePerGacha float64              `json:"discountedPricePerGacha"`
	GameTitle               *GameTitle           `json:"gameTitle,omitempty" gorm:"constraint:OnDelete:CASCADE;"`
	GameTitleID             uint                 `json:"gameTitleId,omitempty"`
	Translations            []PricingTranslation `json:"translations" gorm:"constraint:OnDelete:CASCADE;"`
	PricingTranslatable     `gorm:"-"`
}

type PricingTranslatable struct {
	Name string `json:"name"`
}

type PricingTranslation struct {
	ID        uint     `json:"id"`
	Language  string   `json:"language"`
	Pricing   *Pricing `json:"-"`
	PricingID uint     `json:"-"`
	PricingTranslatable
}

func (pricing Pricing) GetLanguageHolders() []LanguageHolder {
	var languageHolders []LanguageHolder
	for i := 0; i < len(pricing.Translations); i++ {
		languageHolders = append(languageHolders, LanguageHolder{
			GetLanguage: func(i int) func() string {
				return func() string {
					return pricing.Translations[i].Language
				}
			}(i),
		})
	}
	return languageHolders
}

type Policies struct {
	ID                   uint                  `json:"id"`
	Pity                 bool                  `json:"pity"`
	PityTrigger          int                   `json:"pityTrigger"`
	PityItem             *Item                 `json:"pityItem,omitempty" gorm:"constraint:OnDelete:CASCADE;"`
	PityItemID           uint                  `json:"pityItemId,omitempty"`
	GameTitle            *GameTitle            `json:"gameTitle,omitempty" gorm:"constraint:OnDelete:CASCADE;"`
	GameTitleID          uint                  `json:"gameTitleId,omitempty"`
	Translations         []PoliciesTranslation `json:"translations" gorm:"constraint:OnDelete:CASCADE;"`
	PoliciesTranslatable `gorm:"-"`
}

type PoliciesTranslatable struct {
	Name string `json:"name"`
}

type PoliciesTranslation struct {
	ID         uint      `json:"id"`
	Language   string    `json:"language"`
	Policies   *Policies `json:"-"`
	PoliciesID uint      `json:"-"`
	PoliciesTranslatable
}

func (policies Policies) GetLanguageHolders() []LanguageHolder {
	var languageHolders []LanguageHolder
	for i := 0; i < len(policies.Translations); i++ {
		languageHolders = append(languageHolders, LanguageHolder{
			GetLanguage: func(i int) func() string {
				return func() string {
					return policies.Translations[i].Language
				}
			}(i),
		})
	}
	return languageHolders
}

type Plan struct {
	ID                   uint              `json:"id"`
	Budget               float64           `json:"budget"`
	MaxConsecutiveGachas int               `json:"maxConsecutiveGachas"`
	ItemGoals            bool              `json:"itemGoals"`
	WantedItemsJSON      datatypes.JSON    `json:"wantedItemsJSON" gorm:"column:wanted_items"`
	WantedItems          []ItemWithNumber  `json:"wantedItems" gorm:"-"`
	TierGoals            bool              `json:"tierGoals"`
	WantedTiersJSON      datatypes.JSON    `json:"wantedTiersJSON" gorm:"column:wanted_tiers"`
	WantedTiers          []TierWithNumber  `json:"wantedTiers" gorm:"-"`
	GameTitle            *GameTitle        `json:"gameTitle,omitempty" gorm:"constraint:OnDelete:CASCADE;"`
	GameTitleID          uint              `json:"gameTitleId,omitempty"`
	Translations         []PlanTranslation `json:"translations" gorm:"constraint:OnDelete:CASCADE;"`
	PlanTranslatable     `gorm:"-"`
}

type PlanTranslatable struct {
	Name string `json:"name"`
}

type PlanTranslation struct {
	ID       uint   `json:"id"`
	Language string `json:"language"`
	Plan     *Plan  `json:"-"`
	PlanID   uint   `json:"-"`
	PlanTranslatable
}

func (plan Plan) GetLanguageHolders() []LanguageHolder {
	var languageHolders []LanguageHolder
	for i := 0; i < len(plan.Translations); i++ {
		languageHolders = append(languageHolders, LanguageHolder{
			GetLanguage: func(i int) func() string {
				return func() string {
					return plan.Translations[i].Language
				}
			}(i),
		})
	}
	return languageHolders
}

type Result struct {
	ID            uint           `json:"id"`
	UserID        string         `json:"userID" gorm:"index;notNull"`
	Public        bool           `json:"public"`
	Request       datatypes.JSON `json:"request,omitempty"`
	ItemIDs       datatypes.JSON `json:"itemIDs"`
	GoalsAchieved bool           `json:"goalsAchieved"`
	MoneySpent    float64        `json:"moneySpent"`
	Items         datatypes.JSON `json:"-"`
	ItemsResponse []Item         `json:"items,omitempty" gorm:"-"`
	Time          time.Time      `json:"time"`
	GameTitle     *GameTitle     `json:"gameTitle" gorm:"constraint:OnDelete:CASCADE;"`
	GameTitleID   uint           `json:"-"`
}

type GachaRequest struct {
	GameTitle     GameTitle `json:"gameTitle"`
	Tiers         []Tier    `json:"tiers"`
	ItemsIncluded bool      `json:"itemsIncluded"`
	Pricing       Pricing   `json:"pricing"`
	Policies      Policies  `json:"policies"`
	Plan          Plan      `json:"plan"`
}

func SetupDB(dsn string) {
	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		panic(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	err = db.AutoMigrate(
		&GameTitle{},
		&GameTitleTranslation{},
		&Item{},
		&ItemTranslation{},
		&Tier{},
		&TierTranslation{},
		&Pricing{},
		&PricingTranslation{},
		&Policies{},
		&PoliciesTranslation{},
		&Plan{},
		&PlanTranslation{},
		&Result{},
	)
	if err != nil {
		panic(err)
	}
	DB = db
}
