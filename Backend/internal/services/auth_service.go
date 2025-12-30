package services

import (
	"Backend/internal/models"
	"Backend/internal/repositories"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo *repositories.UserRepository
}

func NewAuthService(userRepo *repositories.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

// RegisterRequest ユーザー登録リクエスト
type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	Name        string `json:"name"`
	TargetLevel string `json:"target_level"`
}

// LoginRequest ログインリクエスト
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateProfileRequest プロフィール更新リクエスト
type UpdateProfileRequest struct {
	UserID      uint   `json:"user_id"`
	Name        string `json:"name"`
	TargetLevel string `json:"target_level"`
}

// AuthResponse 認証レスポンス
type AuthResponse struct {
	UserID      uint   `json:"user_id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	IsGuest     bool   `json:"is_guest"`
	TargetLevel string `json:"target_level"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Token       string `json:"token,omitempty"` // 将来的なトークン認証用
}

// Register 新規ユーザー登録
func (s *AuthService) Register(req RegisterRequest) (*AuthResponse, error) {
	// バリデーション
	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email and password are required")
	}
	if req.TargetLevel == "" {
		req.TargetLevel = "新卒"
	}
	if req.TargetLevel != "新卒" && req.TargetLevel != "中途" {
		return nil, errors.New("target_level must be '新卒' or '中途'")
	}

	// 既存ユーザーチェック
	existingUser, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, errors.New("email already exists")
	}

	// パスワードハッシュ化
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// ユーザー作成
	user := &models.User{
		Email:       req.Email,
		Password:    string(hashedPassword),
		Name:        req.Name,
		IsGuest:     false,
		TargetLevel: req.TargetLevel,
	}

	if err := s.userRepo.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &AuthResponse{
		UserID:      user.ID,
		Email:       user.Email,
		Name:        user.Name,
		IsGuest:     user.IsGuest,
		TargetLevel: user.TargetLevel,
	}, nil
}

// Login ログイン処理
func (s *AuthService) Login(req LoginRequest) (*AuthResponse, error) {
	// バリデーション
	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email and password are required")
	}

	// ユーザー取得
	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("invalid email or password")
	}

	// ゲストユーザーはログイン不可
	if user.IsGuest {
		return nil, errors.New("guest users cannot login")
	}

	// パスワード検証
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	return &AuthResponse{
		UserID:      user.ID,
		Email:       user.Email,
		Name:        user.Name,
		IsGuest:     user.IsGuest,
		TargetLevel: user.TargetLevel,
		AvatarURL:   user.AvatarURL,
	}, nil
}

// CreateGuestUser ゲストユーザー作成
func (s *AuthService) CreateGuestUser() (*AuthResponse, error) {
	// ランダムなゲストID生成
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random ID: %w", err)
	}
	guestID := base64.URLEncoding.EncodeToString(randomBytes)

	user := &models.User{
		Email:       fmt.Sprintf("guest_%s@temp.local", guestID),
		Password:    "", // ゲストユーザーはパスワード不要
		Name:        fmt.Sprintf("Guest_%s", guestID[:8]),
		IsGuest:     true,
		TargetLevel: "未設定",
	}

	if err := s.userRepo.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create guest user: %w", err)
	}

	return &AuthResponse{
		UserID:      user.ID,
		Email:       user.Email,
		Name:        user.Name,
		IsGuest:     user.IsGuest,
		TargetLevel: user.TargetLevel,
		AvatarURL:   user.AvatarURL,
	}, nil
}

// GetUser ユーザー情報取得
func (s *AuthService) GetUser(userID uint) (*AuthResponse, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	return &AuthResponse{
		UserID:      user.ID,
		Email:       user.Email,
		Name:        user.Name,
		IsGuest:     user.IsGuest,
		TargetLevel: user.TargetLevel,
	}, nil
}

// UpdateProfile ユーザープロフィール更新
func (s *AuthService) UpdateProfile(req UpdateProfileRequest) (*AuthResponse, error) {
	if req.UserID == 0 {
		return nil, errors.New("user_id is required")
	}
	if req.TargetLevel != "" && req.TargetLevel != "新卒" && req.TargetLevel != "中途" {
		return nil, errors.New("target_level must be '新卒' or '中途'")
	}

	user, err := s.userRepo.GetUserByID(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.TargetLevel != "" {
		user.TargetLevel = req.TargetLevel
	}

	if err := s.userRepo.UpdateUser(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &AuthResponse{
		UserID:      user.ID,
		Email:       user.Email,
		Name:        user.Name,
		IsGuest:     user.IsGuest,
		TargetLevel: user.TargetLevel,
		AvatarURL:   user.AvatarURL,
	}, nil
}
