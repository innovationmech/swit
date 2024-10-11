package model

import (
	"time"

	"github.com/google/uuid"
)

type Token struct {
	ID               uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	UserID           uuid.UUID `json:"user_id" gorm:"type:char(36);not null"`
	AccessToken      string    `json:"access_token" gorm:"type:text;not null"`
	RefreshToken     string    `json:"refresh_token" gorm:"type:text;not null"`
	AccessExpiresAt  time.Time `json:"access_expires_at" gorm:"type:timestamp;not null"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at" gorm:"type:timestamp;not null"`
	IsValid          bool      `json:"is_valid" gorm:"type:boolean;not null;default:true"`
	CreatedAt        time.Time `json:"created_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime"`
}

// TableName 指定数据库表名
func (Token) TableName() string {
	return "tokens"
}
