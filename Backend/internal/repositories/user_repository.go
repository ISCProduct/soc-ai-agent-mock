package repositories

import (
	"Backend/internal/models"
	"errors"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser 新規ユーザー作成
func (r *UserRepository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

// GetUserByEmail メールアドレスでユーザー取得
func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByID IDでユーザー取得
func (r *UserRepository) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// ListUsers ユーザー一覧を取得
func (r *UserRepository) ListUsers() ([]models.User, error) {
	var users []models.User
	err := r.db.Order("created_at desc").Find(&users).Error
	return users, err
}

// ListUsersPaged ページング付きユーザー一覧を取得
func (r *UserRepository) ListUsersPaged(limit, offset int, query string) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	q := r.db.Model(&models.User{})
	if query != "" {
		like := "%" + query + "%"
		q = q.Where("name LIKE ? OR email LIKE ? OR school_name LIKE ?", like, like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at desc").Limit(limit).Offset(offset).Find(&users).Error
	return users, total, err
}

// UpdateUser ユーザー情報更新
func (r *UserRepository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}

// DeleteUser ユーザー削除
func (r *UserRepository) DeleteUser(id uint) error {
	return r.db.Delete(&models.User{}, id).Error
}

// GetUserByVerificationToken メール認証トークンでユーザー取得
func (r *UserRepository) GetUserByVerificationToken(token string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email_verification_token = ?", token).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByOAuth OAuth情報でユーザー取得
func (r *UserRepository) GetUserByOAuth(provider, oauthID string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("oauth_provider = ? AND oauth_id = ?", provider, oauthID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
