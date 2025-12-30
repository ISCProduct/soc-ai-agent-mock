package models

import "time"

// User ユーザー情報
type User struct {
	ID            uint   `gorm:"primaryKey"`
	Email         string `gorm:"size:255;uniqueIndex;not null"`
	Password      string `gorm:"size:255"` // ハッシュ化されたパスワード (OAuth時は空)
	Name          string `gorm:"size:100"`
	IsGuest       bool   `gorm:"default:false"`                  // ゲストユーザーフラグ
	TargetLevel   string `gorm:"size:20;default:'新卒'"`         // 新卒 or 中途
	OAuthProvider string `gorm:"size:50;column:oauth_provider"`  // OAuth提供者 (google, github, など)
	OAuthID       string `gorm:"size:255;index;column:oauth_id"` // OAuth提供者のユーザーID
	AvatarURL     string `gorm:"size:500;column:avatar_url"`     // プロフィール画像URL
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
