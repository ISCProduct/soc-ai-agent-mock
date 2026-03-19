package entity

import "time"

// PendingRegistration メールアドレス確認待ちの仮登録エンティティ
type PendingRegistration struct {
	Token     string
	Email     string
	ExpiresAt time.Time
	CreatedAt time.Time
}
