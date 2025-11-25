package models

import "time"

// User ユーザー情報
type User struct {
	ID            uint   `gorm:"primaryKey"`
	Email         string `gorm:"size:255;uniqueIndex;not null"`
	Password      string `gorm:"size:255"` // ハッシュ化されたパスワード (OAuth時は空)
	Name          string `gorm:"size:100"`
	IsGuest       bool   `gorm:"default:false"`  // ゲストユーザーフラグ
	OAuthProvider string `gorm:"size:50"`        // OAuth提供者 (google, github, など)
	OAuthID       string `gorm:"size:255;index"` // OAuth提供者のユーザーID
	AvatarURL     string `gorm:"size:500"`       // プロフィール画像URL
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
