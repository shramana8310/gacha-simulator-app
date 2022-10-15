package model

import (
	"log"
	"os"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type GameTitle struct {
	ID           uint
	Slug         string `gorm:"size:256;unique;index;notNull"`
	ImageURL     string
	DisplayOrder uint
	Translations []GameTitleTranslation `gorm:"constraint:OnDelete:CASCADE;"`
}

type GameTitleTranslation struct {
	ID          uint
	Language    string
	Name        string
	ShortName   string
	Description string
	GameTitle   *GameTitle
	GameTitleID uint
}

type Tier struct {
	ID           uint
	Ratio        int
	Items        []Item     `gorm:"constraint:OnDelete:CASCADE;"`
	GameTitle    *GameTitle `gorm:"constraint:OnDelete:CASCADE;"`
	GameTitleID  uint
	ImageURL     string
	Translations []TierTranslation `gorm:"constraint:OnDelete:CASCADE;"`
}

type TierTranslation struct {
	ID        uint
	Name      string
	ShortName string
	Language  string
	Tier      *Tier
	TierID    uint
}

type Item struct {
	ID           uint
	Ratio        int
	ImageURL     string
	Tier         *Tier
	TierID       uint
	Translations []ItemTranslation `gorm:"constraint:OnDelete:CASCADE;"`
}

type ItemTranslation struct {
	ID           uint
	Name         string `gorm:"index"`
	ShortName    string `gorm:"index"`
	ShortNameAlt string `gorm:"index"`
	Language     string
	Item         *Item
	ItemID       uint
}

type Pricing struct {
	ID                      uint
	PricePerGacha           float64
	Discount                bool
	DiscountTrigger         int
	DiscountedPricePerGacha float64
	GameTitle               *GameTitle `gorm:"constraint:OnDelete:CASCADE;"`
	GameTitleID             uint
	Translations            []PricingTranslation `gorm:"constraint:OnDelete:CASCADE;"`
}

type PricingTranslation struct {
	ID        uint
	Name      string
	Language  string
	Pricing   *Pricing
	PricingID uint
}

type Policies struct {
	ID           uint
	Pity         bool
	PityTrigger  int
	PityItem     *Item `gorm:"constraint:OnDelete:CASCADE;"`
	PityItemID   *uint
	GameTitle    *GameTitle `gorm:"constraint:OnDelete:CASCADE;"`
	GameTitleID  uint
	Translations []PoliciesTranslation `gorm:"constraint:OnDelete:CASCADE;"`
}

type PoliciesTranslation struct {
	ID         uint
	Name       string
	Language   string
	Policies   *Policies
	PoliciesID uint
}

type Plan struct {
	ID                   uint
	Budget               float64
	MaxConsecutiveGachas int
	ItemGoals            bool
	WantedItemsJSON      datatypes.JSON `gorm:"column:wanted_items"`
	TierGoals            bool
	WantedTiersJSON      datatypes.JSON `gorm:"column:wanted_tiers"`
	GameTitle            *GameTitle     `gorm:"constraint:OnDelete:CASCADE;"`
	GameTitleID          uint
	Translations         []PlanTranslation `gorm:"constraint:OnDelete:CASCADE;"`
}

type PlanTranslation struct {
	ID       uint
	Name     string
	Language string
	Plan     *Plan
	PlanID   uint
}

type Preset struct {
	ID           uint
	GameTitle    *GameTitle `gorm:"constraint:OnDelete:CASCADE;"`
	GameTitleID  uint
	Pricing      *Pricing `gorm:"constraint:OnDelete:CASCADE;"`
	PricingID    *uint
	Policies     *Policies `gorm:"constraint:OnDelete:CASCADE;"`
	PoliciesID   *uint
	Plan         *Plan `gorm:"constraint:OnDelete:CASCADE;"`
	PlanID       *uint
	Translations []PresetTranslation `gorm:"constraint:OnDelete:CASCADE;"`
}

type PresetTranslation struct {
	ID          uint
	Name        string
	Description string
	Language    string
	Preset      *Preset
	PresetID    uint
}

type Result struct {
	ID            uint
	UserID        string `gorm:"index;notNull"`
	Public        bool
	Request       datatypes.JSON
	ItemIDs       datatypes.JSON
	GoalsAchieved bool
	MoneySpent    float64
	Time          time.Time
	GameTitle     *GameTitle `gorm:"constraint:OnDelete:CASCADE;"`
	GameTitleID   uint
}

var DB *gorm.DB

func SetupDB(dsn string) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold: time.Second,
				LogLevel:      logger.Info,
				Colorful:      true,
			}),
	})
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
		&Preset{},
		&PresetTranslation{},
		&Result{},
	)
	if err != nil {
		panic(err)
	}
	DB = db
}

type TranslationHolder interface {
	GetLanguageHolders() []LanguageHolder
}

type LanguageHolder struct {
	GetLanguage func() string
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

func (preset Preset) GetLanguageHolders() []LanguageHolder {
	var languageHolders []LanguageHolder
	for i := 0; i < len(preset.Translations); i++ {
		languageHolders = append(languageHolders, LanguageHolder{
			GetLanguage: func(i int) func() string {
				return func() string {
					return preset.Translations[i].Language
				}
			}(i),
		})
	}
	return languageHolders
}
