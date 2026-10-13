package models

import (
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
	UpdatedAt           time.Time `json:"updated_at"`
}

// DefaultCoursePricing provides fallback prices when database record is not yet present.
var DefaultCoursePricing = CoursePricing{
	AcceleratorPrice:    6900.00,
	AcceleratorOldPrice: 9900.00,
	VIPPrice:            14900.00,
	VIPOldPrice:         19900.00,
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
	if pricing.AcceleratorOldPrice <= 0 {
		pricing.AcceleratorOldPrice = 9900.00
	}
	if pricing.VIPPrice <= 0 {
		pricing.VIPPrice = 14900.00
	}
	if pricing.VIPOldPrice <= 0 {
		pricing.VIPOldPrice = 19900.00
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
		p.UpdatedAt = time.Now()
		if createErr := db.Create(&p).Error; createErr != nil {
			return p, createErr
		}
		return p, nil
	}
	existing.AcceleratorPrice = p.AcceleratorPrice
	existing.AcceleratorOldPrice = p.AcceleratorOldPrice
	existing.VIPPrice = p.VIPPrice
	existing.VIPOldPrice = p.VIPOldPrice
	existing.UpdatedAt = time.Now()
	if saveErr := db.Save(&existing).Error; saveErr != nil {
		return p, saveErr
	}
	return existing, nil
}
