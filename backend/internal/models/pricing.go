package models

import (
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

// CoursePricing stores the dynamic tier pricing configured via the admin panel.
type CoursePricing struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	AcceleratorPrice    float64   `gorm:"type:numeric(10,2);default:6900.00" json:"accelerator_price"`
	AcceleratorOldPrice float64   `gorm:"type:numeric(10,2);default:9900.00" json:"accelerator_old_price"`
	VIPPrice            float64   `gorm:"type:numeric(10,2);default:14900.00" json:"vip_price"`
	VIPOldPrice         float64   `gorm:"type:numeric(10,2);default:19900.00" json:"vip_old_price"`
	BankName            string    `gorm:"size:128;default:'БСПБ (Банк «Санкт-Петербург»)'" json:"bank_name"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// DefaultCoursePricing provides fallback prices when database record is not yet present.
var DefaultCoursePricing = CoursePricing{
	AcceleratorPrice:    6900.00,
	AcceleratorOldPrice: 9900.00,
	VIPPrice:            14900.00,
	VIPOldPrice:         19900.00,
	BankName:            "БСПБ (Банк «Санкт-Петербург»)",
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
	if pricing.AcceleratorPrice <= 0 {
		pricing.AcceleratorPrice = 6900.00
	}
	if pricing.AcceleratorOldPrice <= pricing.AcceleratorPrice {
		pricing.AcceleratorOldPrice = math.Round((pricing.AcceleratorPrice*1.4)/100) * 100
	}
	if pricing.VIPPrice <= 0 {
		pricing.VIPPrice = 14900.00
	}
	if pricing.VIPOldPrice <= pricing.VIPPrice {
		pricing.VIPOldPrice = math.Round((pricing.VIPPrice*1.4)/100) * 100
	}
	// Migrate legacy Ozon Bank or empty bank to BSPB
	if pricing.BankName == "" || strings.Contains(strings.ToLower(pricing.BankName), "озон") || strings.Contains(strings.ToLower(pricing.BankName), "ozon") {
		pricing.BankName = "БСПБ (Банк «Санкт-Петербург»)"
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
		if strings.TrimSpace(p.BankName) == "" {
			p.BankName = "БСПБ (Банк «Санкт-Петербург»)"
		}
		p.UpdatedAt = time.Now()
		if createErr := db.Create(&p).Error; createErr != nil {
			return p, createErr
		}
		return p, nil
	}
	if p.AcceleratorPrice <= 0 {
		p.AcceleratorPrice = 6900.00
	}
	if p.AcceleratorOldPrice <= p.AcceleratorPrice {
		p.AcceleratorOldPrice = math.Round((p.AcceleratorPrice*1.4)/100) * 100
	}
	if p.VIPPrice <= 0 {
		p.VIPPrice = 14900.00
	}
	if p.VIPOldPrice <= p.VIPPrice {
		p.VIPOldPrice = math.Round((p.VIPPrice*1.4)/100) * 100
	}
	if strings.TrimSpace(p.BankName) == "" {
		p.BankName = "БСПБ (Банк «Санкт-Петербург»)"
	}

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
