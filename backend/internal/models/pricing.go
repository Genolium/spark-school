package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Default course prices (RUB). This is the single backend source for fallback
// prices used when the course_pricings row is missing or holds a non-positive
// value. Real prices live in the DB and are edited in the admin panel; keep
// these in sync with frontend/src/lib/pricing-defaults.ts (enforced by
// TestDefaultCoursePricingMatchesFrontendDefaults).
const (
	DefaultBasicPrice          = 3333.00
	DefaultBasicOldPrice       = 5900.00
	DefaultAcceleratorPrice    = 10990.00
	DefaultAcceleratorOldPrice = 15400.00
	DefaultVIPPrice            = 19990.00
	DefaultVIPOldPrice         = 28000.00
	DefaultBankName            = "БСПБ (Банк «Санкт-Петербург»)"
)

// DefaultTierPrice returns the default list price for a tier ("basic",
// "accelerator", "vip", including Russian aliases). Unknown tiers fall back to
// the accelerator price, the flagship tier.
func DefaultTierPrice(tier string) float64 {
	return DefaultCoursePricing.TierPrice(tier)
}

// TierPrice returns the list price of the given tier from this pricing record.
// Unknown tiers resolve to the accelerator price.
func (p CoursePricing) TierPrice(tier string) float64 {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "basic", "базовый":
		return p.BasicPrice
	case "vip":
		return p.VIPPrice
	default:
		return p.AcceleratorPrice
	}
}

// DefaultUpgradeDiff returns the default surcharge for upgrading between two
// tiers, computed from the default prices (never negative).
func DefaultUpgradeDiff(fromTier, toTier string) float64 {
	diff := DefaultTierPrice(toTier) - DefaultTierPrice(fromTier)
	if diff < 0 {
		return 0
	}
	return diff
}

// CoursePricing stores the dynamic tier pricing configured via the admin panel.
//
// The gorm `default:` tags below are not applied to production: the schema is
// managed by Goose SQL migrations (no AutoMigrate at runtime). They only shape
// tables created by AutoMigrate in tests and must mirror the Default* constants.
type CoursePricing struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	BasicPrice          float64   `gorm:"type:numeric(10,2);default:3333.00" json:"basic_price"`
	BasicOldPrice       float64   `gorm:"type:numeric(10,2);default:5900.00" json:"basic_old_price"`
	AcceleratorPrice    float64   `gorm:"type:numeric(10,2);default:10990.00" json:"accelerator_price"`
	AcceleratorOldPrice float64   `gorm:"type:numeric(10,2);default:15400.00" json:"accelerator_old_price"`
	VIPPrice            float64   `gorm:"type:numeric(10,2);default:19990.00" json:"vip_price"`
	VIPOldPrice         float64   `gorm:"type:numeric(10,2);default:28000.00" json:"vip_old_price"`
	BankName            string    `gorm:"size:128;default:'БСПБ (Банк «Санкт-Петербург»)'" json:"bank_name"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// DefaultCoursePricing provides fallback prices when database record is not yet present.
var DefaultCoursePricing = CoursePricing{
	BasicPrice:          DefaultBasicPrice,
	BasicOldPrice:       DefaultBasicOldPrice,
	AcceleratorPrice:    DefaultAcceleratorPrice,
	AcceleratorOldPrice: DefaultAcceleratorOldPrice,
	VIPPrice:            DefaultVIPPrice,
	VIPOldPrice:         DefaultVIPOldPrice,
	BankName:            DefaultBankName,
}

// GetCoursePricing returns the latest configured course pricing from DB or default fallback.
func GetCoursePricing(db *gorm.DB) CoursePricing {
	if db == nil {
		return DefaultCoursePricing
	}
	var pricing CoursePricing
	if err := db.Order("id ASC").First(&pricing).Error; err != nil {
		pricing = DefaultCoursePricing
		pricing.UpdatedAt = time.Now()
		_ = db.Create(&pricing).Error
		return pricing
	}
	if pricing.BasicPrice <= 0 {
		pricing.BasicPrice = DefaultBasicPrice
	}
	if pricing.BasicOldPrice <= 0 {
		pricing.BasicOldPrice = pricing.BasicPrice
	}
	if pricing.AcceleratorPrice <= 0 {
		pricing.AcceleratorPrice = DefaultAcceleratorPrice
	}
	if pricing.AcceleratorOldPrice <= 0 {
		pricing.AcceleratorOldPrice = pricing.AcceleratorPrice
	}
	if pricing.VIPPrice <= 0 {
		pricing.VIPPrice = DefaultVIPPrice
	}
	if pricing.VIPOldPrice <= 0 {
		pricing.VIPOldPrice = pricing.VIPPrice
	}
	// Migrate legacy Ozon Bank or empty bank to BSPB
	if pricing.BankName == "" || strings.Contains(strings.ToLower(pricing.BankName), "озон") || strings.Contains(strings.ToLower(pricing.BankName), "ozon") {
		pricing.BankName = DefaultBankName
		_ = db.Model(&pricing).Update("bank_name", pricing.BankName).Error
	}
	return pricing
}

// UpdateCoursePricing updates or creates course pricing record in DB.
func UpdateCoursePricing(db *gorm.DB, p CoursePricing) (CoursePricing, error) {
	if db == nil {
		return p, nil
	}
	var existing CoursePricing
	err := db.Order("id ASC").First(&existing).Error
	if err != nil {
		if p.BasicPrice <= 0 {
			p.BasicPrice = DefaultBasicPrice
		}
		if p.BasicOldPrice <= 0 {
			p.BasicOldPrice = p.BasicPrice
		}
		if p.AcceleratorPrice <= 0 {
			p.AcceleratorPrice = DefaultAcceleratorPrice
		}
		if p.AcceleratorOldPrice <= 0 {
			p.AcceleratorOldPrice = p.AcceleratorPrice
		}
		if strings.TrimSpace(p.BankName) == "" {
			p.BankName = DefaultBankName
		}
		p.UpdatedAt = time.Now()
		if createErr := db.Create(&p).Error; createErr != nil {
			return p, createErr
		}
		return p, nil
	}
	if p.BasicPrice <= 0 {
		p.BasicPrice = DefaultBasicPrice
	}
	if p.BasicOldPrice <= 0 {
		p.BasicOldPrice = p.BasicPrice
	}
	if p.AcceleratorPrice <= 0 {
		p.AcceleratorPrice = DefaultAcceleratorPrice
	}
	if p.AcceleratorOldPrice <= 0 {
		p.AcceleratorOldPrice = p.AcceleratorPrice
	}
	if p.VIPPrice <= 0 {
		p.VIPPrice = DefaultVIPPrice
	}
	if p.VIPOldPrice <= 0 {
		p.VIPOldPrice = p.VIPPrice
	}
	if strings.TrimSpace(p.BankName) == "" {
		p.BankName = DefaultBankName
	}

	existing.BasicPrice = p.BasicPrice
	existing.BasicOldPrice = p.BasicOldPrice
	existing.AcceleratorPrice = p.AcceleratorPrice
	existing.AcceleratorOldPrice = p.AcceleratorOldPrice
	existing.VIPPrice = p.VIPPrice
	existing.VIPOldPrice = p.VIPOldPrice
	existing.BankName = strings.TrimSpace(p.BankName)
	existing.UpdatedAt = time.Now()
	if saveErr := db.Save(&existing).Error; saveErr != nil {
		return p, saveErr
	}
	return existing, nil
}
