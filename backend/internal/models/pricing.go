package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// CoursePricing stores the dynamic tier pricing configured via the admin panel.
type CoursePricing struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	BasicPrice          float64   `gorm:"type:numeric(10,2);default:3990.00" json:"basic_price"`
	BasicOldPrice       float64   `gorm:"type:numeric(10,2);default:5900.00" json:"basic_old_price"`
	AcceleratorPrice    float64   `gorm:"type:numeric(10,2);default:10900.00" json:"accelerator_price"`
	AcceleratorOldPrice float64   `gorm:"type:numeric(10,2);default:10900.00" json:"accelerator_old_price"`
	VIPPrice            float64   `gorm:"type:numeric(10,2);default:14900.00" json:"vip_price"`
	VIPOldPrice         float64   `gorm:"type:numeric(10,2);default:19900.00" json:"vip_old_price"`
	BankName            string    `gorm:"size:128;default:'БСПБ (Банк «Санкт-Петербург»)'" json:"bank_name"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// DefaultCoursePricing provides fallback prices when database record is not yet present.
var DefaultCoursePricing = CoursePricing{
	BasicPrice:          3990.00,
	BasicOldPrice:       5900.00,
	AcceleratorPrice:    10900.00,
	AcceleratorOldPrice: 10900.00,
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
	if pricing.BasicPrice <= 0 {
		pricing.BasicPrice = 3990.00
	}
	if pricing.BasicOldPrice <= 0 {
		pricing.BasicOldPrice = pricing.BasicPrice
	}
	if pricing.AcceleratorPrice <= 0 {
		pricing.AcceleratorPrice = 10900.00
	}
	if pricing.AcceleratorOldPrice <= 0 {
		pricing.AcceleratorOldPrice = 10900.00
	}
	if pricing.VIPPrice <= 0 {
		pricing.VIPPrice = 14900.00
	}
	if pricing.VIPOldPrice <= 0 {
		pricing.VIPOldPrice = pricing.VIPPrice
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
		if p.BasicPrice <= 0 {
			p.BasicPrice = 3990.00
		}
		if p.BasicOldPrice <= 0 {
			p.BasicOldPrice = p.BasicPrice
		}
		if p.AcceleratorPrice <= 0 {
			p.AcceleratorPrice = 10900.00
		}
		if p.AcceleratorOldPrice <= 0 {
			p.AcceleratorOldPrice = 10900.00
		}
		if strings.TrimSpace(p.BankName) == "" {
			p.BankName = "БСПБ (Банк «Санкт-Петербург»)"
		}
		p.UpdatedAt = time.Now()
		if createErr := db.Create(&p).Error; createErr != nil {
			return p, createErr
		}
		return p, nil
	}
	if p.BasicPrice <= 0 {
		p.BasicPrice = 3990.00
	}
	if p.BasicOldPrice <= 0 {
		p.BasicOldPrice = p.BasicPrice
	}
	if p.AcceleratorPrice <= 0 {
		p.AcceleratorPrice = 10900.00
	}
	if p.AcceleratorOldPrice <= 0 {
		p.AcceleratorOldPrice = 10900.00
	}
	if p.VIPPrice <= 0 {
		p.VIPPrice = 14900.00
	}
	if p.VIPOldPrice <= 0 {
		p.VIPOldPrice = p.VIPPrice
	}
	if strings.TrimSpace(p.BankName) == "" {
		p.BankName = "БСПБ (Банк «Санкт-Петербург»)"
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
