package models

import "time"

// User ユーザー情報
type User struct {
	ID        uint   `gorm:"primaryKey"`
	Email     string `gorm:"size:255;uniqueIndex;not null"`
	Password  string `gorm:"size:255;not null"` // ハッシュ化されたパスワード
	Name      string `gorm:"size:100"`
	IsGuest   bool   `gorm:"default:false"` // ゲストユーザーフラグ
	CreatedAt time.Time
	UpdatedAt time.Time
}
