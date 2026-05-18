package models

import "time"

type Route struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Path      string    `json:"path" gorm:"not null;uniqueIndex"`
	TargetURL string    `json:"target_url" gorm:"not null"`
	Methods   string    `json:"methods" gorm:"not null"`
	IsActive  bool      `json:"is_active" gorm:"not null;default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
