// Package repository defines repository interfaces for the domain layer.
// Concrete implementations live in internal/repositories/.
package repository

import "Backend/internal/models"

// UserRepository はユーザー永続化の抽象インターフェース。
type UserRepository interface {
	CreateUser(user *models.User) error
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id uint) (*models.User, error)
	ListUsers() ([]models.User, error)
	ListUsersPaged(limit, offset int, query string) ([]models.User, int64, error)
	UpdateUser(user *models.User) error
	DeleteUser(id uint) error
	GetUserByVerificationToken(token string) (*models.User, error)
	GetUserByPasswordResetToken(token string) (*models.User, error)
	GetUserByOAuth(provider, oauthID string) (*models.User, error)
}

// PendingRegistrationRepository は仮登録の永続化インターフェース。
type PendingRegistrationRepository interface {
	Create(p *models.PendingRegistration) error
	FindByToken(token string) (*models.PendingRegistration, error)
	DeleteByEmail(email string) error
	DeleteExpired() error
}
