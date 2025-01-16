package models

import (
	"time"
)

type HomeadvisorListingModel struct {
	Url            string `gorm:"primarykey"`
	CompanyUrl     *string
	CompanyName    *string
	State          *string
	City           *string
	PhoneNumber    *string
	ReviewCount    uint `gorm:"default:0"`
	LastReviewDate *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
