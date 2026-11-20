package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// CoursePricing stores the dynamic tier pricing configured via the admin panel.
type CoursePricing struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	BasicPrice            float64    `gorm:"type:numeric(10,2);default:3333.00" json:"basic_price"`
	BasicOldPrice         float64    `gorm:"type:numeric(10,2);default:5900.00" json:"basic_old_price"`
	BasicFeedbackPrice    float64    `gorm:"type:numeric(10,2);default:5000.00" json:"basic_feedback_price"`
	BasicFeedbackOldPrice float64    `gorm:"type:numeric(10,2);default:6900.00" json:"basic_feedback_old_price"`
	AcceleratorPrice      float64    `gorm:"type:numeric(10,2);default:9990.00" json:"accelerator_price"`
	AcceleratorOldPrice   float64    `gorm:"type:numeric(10,2);default:15400.00" json:"accelerator_old_price"`
	VIPPrice              float64    `gorm:"column:vip_price;type:numeric(10,2);default:15990.00" json:"vip_price"`
	VIPOldPrice           float64    `gorm:"column:vip_old_price;type:numeric(10,2);default:20000.00" json:"vip_old_price"`
	BasicEnabled          bool       `gorm:"not null;default:true" json:"basic_enabled"`
	BasicFeedbackEnabled  bool       `gorm:"not null;default:true" json:"basic_feedback_enabled"`
	AcceleratorEnabled    bool       `gorm:"not null;default:true" json:"accelerator_enabled"`
	VIPEnabled            bool       `gorm:"column:vip_enabled;not null;default:true" json:"vip_enabled"`
	SalesCloseAt          *time.Time `json:"sales_close_at"`
	SalesClosed           bool       `gorm:"-" json:"sales_closed"`
	BankName              string     `gorm:"size:128;default:'БСПБ (Банк «Санкт-Петербург»)'" json:"bank_name"`
	UpdatedAt             time.Time  `json:"updated_at"`

	// TierFlagsSet distinguishes an explicitly disabled tier from a legacy
	// CoursePricing value built without the new fields.
	TierFlagsSet bool `gorm:"-" json:"-"`
	// SalesCloseAtSet distinguishes an explicit SalesCloseAt update (including
	// clearing it to nil) from a legacy payload that omitted the field.
	SalesCloseAtSet bool `gorm:"-" json:"-"`
}

// DefaultSalesCloseAt returns the scheduled close of Stage 1 sales:
// 21 November 2026 00:00:00 MSK (UTC+3).
func DefaultSalesCloseAt() *time.Time {
	t, err := time.Parse(time.RFC3339, "2026-11-21T00:00:00+03:00")
	if err != nil {
		return nil
	}
	return &t
}

// DefaultCoursePricing provides fallback prices when database record is not yet present.
var DefaultCoursePricing = CoursePricing{
	BasicPrice:            3333.00,
	BasicOldPrice:         5900.00,
	BasicFeedbackPrice:    5000.00,
	BasicFeedbackOldPrice: 6900.00,
	AcceleratorPrice:      9990.00,
	AcceleratorOldPrice:   15400.00,
	VIPPrice:              15990.00,
	VIPOldPrice:           20000.00,
	BasicEnabled:          true,
	BasicFeedbackEnabled:  true,
	AcceleratorEnabled:    true,
	VIPEnabled:            true,
	SalesCloseAt:          DefaultSalesCloseAt(),
	BankName:              "БСПБ (Банк «Санкт-Петербург»)",
	TierFlagsSet:          true,
}

// IsSalesClosedAt reports whether all new sales are closed at the given time
// (either because SalesCloseAt has passed or because every tier is disabled).
func (p CoursePricing) IsSalesClosedAt(now time.Time) bool {
	if p.SalesCloseAt != nil && !now.Before(*p.SalesCloseAt) {
		return true
	}
	if p.TierFlagsSet && !p.BasicEnabled && !p.BasicFeedbackEnabled && !p.AcceleratorEnabled && !p.VIPEnabled {
		return true
	}
	return false
}

// TierEnabledAt reports whether new purchases of a tier are allowed at now.
func (p CoursePricing) TierEnabledAt(tier string, now time.Time) bool {
	if p.SalesCloseAt != nil && !now.Before(*p.SalesCloseAt) {
		return false
	}
	if !p.TierFlagsSet {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "basic", "базовый":
		return p.BasicEnabled
	case "basic_feedback", "basic_plus":
		return p.BasicFeedbackEnabled
	case "accelerator", "акселератор":
		return p.AcceleratorEnabled
	case "vip":
		return p.VIPEnabled
	default:
		return false
	}
}

// TierEnabled reports whether new purchases of a tier are allowed right now.
// Legacy values without TierFlagsSet keep the previous behaviour (all tiers enabled).
func (p CoursePricing) TierEnabled(tier string) bool {
	return p.TierEnabledAt(tier, time.Now())
}

// FirstEnabledTier returns the first tier that can be purchased, or "".
func (p CoursePricing) FirstEnabledTier() string {
	for _, tier := range []string{"basic", "basic_feedback", "accelerator", "vip"} {
		if p.TierEnabled(tier) {
			return tier
		}
	}
	return ""
}

// GetCoursePricing returns the latest configured course pricing from DB or default fallback.
func GetCoursePricing(db *gorm.DB) CoursePricing {
	if db == nil {
		p := DefaultCoursePricing
		p.SalesClosed = p.IsSalesClosedAt(time.Now())
		return p
	}
	var pricing CoursePricing
	if err := db.Order("id ASC").First(&pricing).Error; err != nil {
		pricing = DefaultCoursePricing
		pricing.UpdatedAt = time.Now()
		_ = db.Create(&pricing).Error
		pricing.SalesClosed = pricing.IsSalesClosedAt(time.Now())
		return pricing
	}
	pricing.TierFlagsSet = true
	if pricing.BasicPrice <= 0 {
		pricing.BasicPrice = 3333.00
	}
	if pricing.BasicOldPrice <= 0 {
		pricing.BasicOldPrice = pricing.BasicPrice
	}
	if pricing.BasicFeedbackPrice <= 0 {
		pricing.BasicFeedbackPrice = 5000.00
	}
	if pricing.BasicFeedbackOldPrice <= 0 {
		pricing.BasicFeedbackOldPrice = 6900.00
	}
	if pricing.AcceleratorPrice <= 0 {
		pricing.AcceleratorPrice = 9990.00
	}
	if pricing.AcceleratorOldPrice <= 0 {
		pricing.AcceleratorOldPrice = 15400.00
	}
	if pricing.VIPPrice <= 0 {
		pricing.VIPPrice = 15990.00
	}
	if pricing.VIPOldPrice <= 0 {
		pricing.VIPOldPrice = pricing.VIPPrice
	}
	// Migrate legacy Ozon Bank or empty bank to BSPB
	if pricing.BankName == "" || strings.Contains(strings.ToLower(pricing.BankName), "озон") || strings.Contains(strings.ToLower(pricing.BankName), "ozon") {
		pricing.BankName = "БСПБ (Банк «Санкт-Петербург»)"
		_ = db.Model(&pricing).Update("bank_name", pricing.BankName).Error
	}
	pricing.SalesClosed = pricing.IsSalesClosedAt(time.Now())
	return pricing
}

// UpdateCoursePricing updates or creates course pricing record in DB.
func UpdateCoursePricing(db *gorm.DB, p CoursePricing) (CoursePricing, error) {
	if db == nil {
		p.SalesClosed = p.IsSalesClosedAt(time.Now())
		return p, nil
	}
	var existing CoursePricing
	err := db.Order("id ASC").First(&existing).Error
	if err != nil {
		if p.BasicPrice <= 0 {
			p.BasicPrice = 3333.00
		}
		if p.BasicOldPrice <= 0 {
			p.BasicOldPrice = p.BasicPrice
		}
		if p.BasicFeedbackPrice <= 0 {
			p.BasicFeedbackPrice = 5000.00
		}
		if p.BasicFeedbackOldPrice <= 0 {
			p.BasicFeedbackOldPrice = 6900.00
		}
		if p.AcceleratorPrice <= 0 {
			p.AcceleratorPrice = 9990.00
		}
		if p.AcceleratorOldPrice <= 0 {
			p.AcceleratorOldPrice = 15400.00
		}
		if strings.TrimSpace(p.BankName) == "" {
			p.BankName = "БСПБ (Банк «Санкт-Петербург»)"
		}
		if !p.TierFlagsSet {
			p.BasicEnabled, p.BasicFeedbackEnabled, p.AcceleratorEnabled, p.VIPEnabled = true, true, true, true
		}
		if !p.SalesCloseAtSet {
			p.SalesCloseAt = DefaultSalesCloseAt()
		}
		basicEnabled, basicFeedbackEnabled, acceleratorEnabled, vipEnabled := p.BasicEnabled, p.BasicFeedbackEnabled, p.AcceleratorEnabled, p.VIPEnabled
		salesCloseAt := p.SalesCloseAt
		p.TierFlagsSet = true
		p.UpdatedAt = time.Now()
		// Select("*") keeps explicitly disabled flags; a plain Create would
		// replace a false value with the column default (true).
		if createErr := db.Select("*").Create(&p).Error; createErr != nil {
			return p, createErr
		}
		// GORM may still apply the column default to a zero bool on Create and
		// write it back into the struct; persist the explicit flags once more.
		if updateErr := db.Model(&p).Updates(map[string]interface{}{
			"basic_enabled":          basicEnabled,
			"basic_feedback_enabled": basicFeedbackEnabled,
			"accelerator_enabled":    acceleratorEnabled,
			"vip_enabled":            vipEnabled,
			"sales_close_at":         salesCloseAt,
		}).Error; updateErr != nil {
			return p, updateErr
		}
		p.BasicEnabled, p.BasicFeedbackEnabled, p.AcceleratorEnabled, p.VIPEnabled = basicEnabled, basicFeedbackEnabled, acceleratorEnabled, vipEnabled
		p.SalesCloseAt = salesCloseAt
		p.SalesClosed = p.IsSalesClosedAt(time.Now())
		return p, nil
	}
	if p.BasicPrice <= 0 {
		p.BasicPrice = 3333.00
	}
	if p.BasicOldPrice <= 0 {
		p.BasicOldPrice = p.BasicPrice
	}
	if p.BasicFeedbackPrice <= 0 {
		p.BasicFeedbackPrice = 5000.00
	}
	if p.BasicFeedbackOldPrice <= 0 {
		p.BasicFeedbackOldPrice = 6900.00
	}
	if p.AcceleratorPrice <= 0 {
		p.AcceleratorPrice = 9990.00
	}
	if p.AcceleratorOldPrice <= 0 {
		p.AcceleratorOldPrice = 15400.00
	}
	if p.VIPPrice <= 0 {
		p.VIPPrice = 15990.00
	}
	if p.VIPOldPrice <= 0 {
		p.VIPOldPrice = p.VIPPrice
	}
	if strings.TrimSpace(p.BankName) == "" {
		p.BankName = "БСПБ (Банк «Санкт-Петербург»)"
	}

	existing.BasicPrice = p.BasicPrice
	existing.BasicOldPrice = p.BasicOldPrice
	existing.BasicFeedbackPrice = p.BasicFeedbackPrice
	existing.BasicFeedbackOldPrice = p.BasicFeedbackOldPrice
	existing.AcceleratorPrice = p.AcceleratorPrice
	existing.AcceleratorOldPrice = p.AcceleratorOldPrice
	existing.VIPPrice = p.VIPPrice
	existing.VIPOldPrice = p.VIPOldPrice
	if p.TierFlagsSet {
		existing.BasicEnabled = p.BasicEnabled
		existing.BasicFeedbackEnabled = p.BasicFeedbackEnabled
		existing.AcceleratorEnabled = p.AcceleratorEnabled
		existing.VIPEnabled = p.VIPEnabled
	}
	if p.SalesCloseAtSet {
		existing.SalesCloseAt = p.SalesCloseAt
	}
	existing.TierFlagsSet = true
	existing.BankName = strings.TrimSpace(p.BankName)
	existing.UpdatedAt = time.Now()
	if saveErr := db.Save(&existing).Error; saveErr != nil {
		return p, saveErr
	}
	existing.SalesClosed = existing.IsSalesClosedAt(time.Now())
	return existing, nil
}
