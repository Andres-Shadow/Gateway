package models

import "time"

type Route struct {
	ID                    uint      `json:"id" gorm:"primaryKey"`
	Path                  string    `json:"path" gorm:"not null;uniqueIndex"`
	TargetURL             string    `json:"target_url" gorm:"not null"`
	Methods               string    `json:"methods" gorm:"not null"`
	IsActive              bool      `json:"is_active" gorm:"not null;default:true"`
	HealthCheckPath       string    `json:"health_check_path,omitempty"`
	RewritePrefixFrom     string    `json:"rewrite_prefix_from,omitempty"`
	RewritePrefixTo       string    `json:"rewrite_prefix_to,omitempty"`
	RequestHeadersSet     string    `json:"request_headers_set,omitempty" gorm:"type:text"`
	RequestHeadersRemove  string    `json:"request_headers_remove,omitempty" gorm:"type:text"`
	ResponseHeadersSet    string    `json:"response_headers_set,omitempty" gorm:"type:text"`
	ResponseHeadersRemove string    `json:"response_headers_remove,omitempty" gorm:"type:text"`
	RequestBodyTransform  string    `json:"request_body_transform,omitempty"`
	ResponseBodyTransform string    `json:"response_body_transform,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}
