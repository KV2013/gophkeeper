package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/victor/gophkeeper/internal/crypto"
	"github.com/victor/gophkeeper/internal/model"
	repoerrors "github.com/victor/gophkeeper/internal/repository/errors"
)

// bcryptCost — стоимость bcrypt-хеширования пароля.
const bcryptCost = bcrypt.DefaultCost

// userRepository — минимальный интерфейс хранилища, необходимый AuthService.
type userRepository interface {
	// CreateUser создаёт нового пользователя.
	CreateUser(ctx context.Context, user *model.User) error
	// GetUserByLogin возвращает пользователя по логину.
	GetUserByLogin(ctx context.Context, login string) (*model.User, error)
}

// AuthService реализует регистрацию и аутентификацию пользователей.
type AuthService struct {
	repo      userRepository
	logger    *zap.Logger
	jwtSecret []byte
	tokenTTL  time.Duration
}

// NewAuthService создаёт сервис аутентификации.
func NewAuthService(repo userRepository, logger *zap.Logger, jwtSecret string, tokenTTL time.Duration) *AuthService {
	return &AuthService{
		repo:      repo,
		logger:    logger,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  tokenTTL,
	}
}

// Register регистрирует нового пользователя и возвращает JWT-токен.
func (s *AuthService) Register(ctx context.Context, login, password string) (string, error) {
	if login == "" || password == "" {
		return "", ErrBadRequest
	}

	salt, err := crypto.NewSalt()
	if err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}

	user := &model.User{
		ID:           uuid.NewString(),
		Login:        login,
		PasswordHash: string(hash),
		Salt:         salt,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		if errors.Is(err, repoerrors.ErrLoginExists) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}

	return s.issueToken(user.ID)
}

// Login аутентифицирует пользователя и возвращает JWT-токен.
func (s *AuthService) Login(ctx context.Context, login, password string) (string, error) {
	user, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, repoerrors.ErrNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return s.issueToken(user.ID)
}

// issueToken выпускает JWT-токен для пользователя.
func (s *AuthService) issueToken(userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
