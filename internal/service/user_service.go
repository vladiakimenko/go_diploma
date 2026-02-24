package service

import (
	"context"
	"errors"
	"time"

	"blog-api/internal/model"
	"blog-api/internal/repository"
	"blog-api/pkg/auth"

	"github.com/gofrs/uuid/v5"
)

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshTokenExpired = errors.New("refresh token expired")
	ErrWeakPassword        = errors.New("weak password")
	ErrTokenGeneration     = errors.New("token generation failed")
	ErrPasswordHash        = errors.New("password hash failed")
)

type UserService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	jwtManager       *auth.JWTManager
	passwordManager  *auth.PasswordManager
}

func NewUserService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtManager *auth.JWTManager,
	passwordManager *auth.PasswordManager,
) *UserService {
	return &UserService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtManager:       jwtManager,
		passwordManager:  passwordManager,
	}
}

func (s *UserService) Register(ctx context.Context, req *model.UserCreateRequest) (*model.TokenResponse, error) {
	var tokenResp *model.TokenResponse
	err := s.userRepo.WithinTransaction(
		ctx,
		func(txCtx context.Context) error {
			exists, err := s.userRepo.ExistsByField(txCtx, "email", req.Email)
			if err != nil {
				logger.Error("failed to check user existance: %v", err)
				return ErrDatabase
			}
			if exists {
				return ErrUserAlreadyExists
			}

			exists, err = s.userRepo.ExistsByField(txCtx, "username", req.Username)
			if err != nil {
				logger.Error("failed to check user existance: %v", err)
				return ErrDatabase
			}
			if exists {
				return ErrUserAlreadyExists
			}

			if err := s.passwordManager.ValidatePasswordStrength(req.Password); err != nil {
				return ErrWeakPassword
			}

			hashedPassword, err := s.passwordManager.HashPassword(txCtx, req.Password)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return err
				}
				logger.Error("failed to hash a password: %v", err)
				return ErrPasswordHash
			}

			user := &model.User{
				Email:        req.Email,
				Username:     req.Username,
				PasswordHash: hashedPassword,
			}

			if err := s.userRepo.Create(txCtx, user); err != nil {
				logger.Error("failed to create a user: %v", err)
				return ErrDatabase
			}

			tokenResp, err = s.createTokenPair(txCtx, user.ID)
			return err
		},
	)
	if err != nil {
		return nil, err
	}
	return tokenResp, nil
}

func (s *UserService) Login(
	ctx context.Context,
	req *model.UserLoginRequest,
) (*model.TokenResponse, error) {
	user, err := s.userRepo.GetByField(ctx, "email", req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			// Do not reveal if email exists or not
			return nil, ErrInvalidCredentials
		}
		logger.Error("failed to fetch a user: %v", err)
		return nil, ErrDatabase
	}

	if !s.passwordManager.CheckPassword(req.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	return s.createTokenPair(ctx, user.ID)
}

func (s *UserService) RefreshToken(ctx context.Context, req *model.RefreshTokenRequest) (*model.TokenResponse, error) {
	var tokenResp *model.TokenResponse

	err := s.userRepo.WithinTransaction(
		ctx,
		func(txCtx context.Context) error {
			tokenUUID, err := uuid.FromString(req.RefreshToken)
			if err != nil {
				return ErrInvalidRefreshToken
			}

			rt, err := s.refreshTokenRepo.GetByValue(txCtx, tokenUUID)
			if err != nil {
				if errors.Is(err, repository.ErrRefreshTokenNotFound) {
					return ErrInvalidRefreshToken
				}
				logger.Error("failed to fetch a refresh token: %v", err)
				return ErrDatabase
			}

			if time.Now().After(rt.ExpiresAt) {
				if err := s.refreshTokenRepo.DeleteByValue(txCtx, tokenUUID); err != nil {
					logger.Error("failed to delete expired refresh token: %v", err)
				}
				return ErrRefreshTokenExpired
			}

			if err := s.refreshTokenRepo.DeleteByValue(txCtx, tokenUUID); err != nil {
				logger.Error("failed to delete old refresh token: %v", err)
				return ErrDatabase
			}

			tokenResp, err = s.createTokenPair(txCtx, rt.UserID)
			return err
		},
	)

	if err != nil {
		return nil, err
	}

	return tokenResp, nil
}

func (s *UserService) createTokenPair(ctx context.Context, userID int) (*model.TokenResponse, error) {
	accessToken, accessExpiresAt, err := s.jwtManager.GenerateToken(ctx, userID)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, ErrTokenGeneration
	}

	refreshToken := &model.RefreshToken{
		Value:     uuid.Must(uuid.NewV4()),
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.jwtManager.RefreshTokenTTL),
	}

	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, ErrDatabase
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &model.TokenResponse{
		AccessToken:        accessToken,
		AccessTokenExpiry:  accessExpiresAt,
		RefreshToken:       refreshToken.Value.String(),
		RefreshTokenExpiry: refreshToken.ExpiresAt,
		User:               user,
	}, nil
}

func (s *UserService) GetByID(ctx context.Context, id int) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, ErrDatabase
	}
	return user, nil
}
