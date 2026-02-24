package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"blog-api/internal/model"
	"blog-api/internal/repository"
	"blog-api/pkg/auth"

	"github.com/gofrs/uuid/v5"
	"golang.org/x/crypto/bcrypt"
)

func setupUserServiceForTest(
	userRepo repository.UserRepository,
	rtRepo repository.RefreshTokenRepository,
) *UserService {
	jwtCfg := &auth.JWTConfig{
		RefreshTokenTTLHours: 1,
	}
	jwtMgr := auth.NewJWTManager(jwtCfg)

	passMgr := auth.NewPasswordManager(&auth.PasswordConfig{
		MinLength:         6,
		Cost:              4,
		CaseShiftRequired: false,
		DigitsRequired:    false,
		SymbolsRequired:   false,
	})

	return NewUserService(userRepo, rtRepo, jwtMgr, passMgr)
}

func TestUserServiceRegister(t *testing.T) {
	ctx := context.Background()
	userRepo := repository.NewInMemoryUserRepo()
	rtRepo := repository.NewInMemoryRefreshTokenRepo()
	svc := setupUserServiceForTest(userRepo, rtRepo)

	req := &model.UserCreateRequest{
		Email:    "test@example.com",
		Username: "tester",
		Password: "StrongPass123!",
	}

	// success
	tokenResp, err := svc.Register(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tokenResp.User.Email != req.Email {
		t.Fatalf("expected email %s, got %s", req.Email, tokenResp.User.Email)
	}

	// existing email
	_, err = svc.Register(ctx, &model.UserCreateRequest{
		Email:    req.Email,
		Username: "newuser",
		Password: "StrongPass123!",
	})
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
	}

	// existing username
	_, err = svc.Register(ctx, &model.UserCreateRequest{
		Email:    "new@example.com",
		Username: req.Username,
		Password: "StrongPass123!",
	})
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestUserService_Login(t *testing.T) {
	ctx := context.Background()
	userRepo := repository.NewInMemoryUserRepo()
	rtRepo := repository.NewInMemoryRefreshTokenRepo()
	svc := setupUserServiceForTest(userRepo, rtRepo)

	hash, _ := bcrypt.GenerateFromPassword([]byte("CorrectPassword12345!"), bcrypt.MinCost)
	user := &model.User{Email: "test@example.com", Username: "tester", PasswordHash: string(hash)}
	userRepo.Create(ctx, user)

	// success
	tokenResp, err := svc.Login(ctx, &model.UserLoginRequest{
		Email:    user.Email,
		Password: "CorrectPassword12345!",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tokenResp.User.Email != user.Email {
		t.Fatalf("expected email %s, got %s", user.Email, tokenResp.User.Email)
	}

	// wrong password
	_, err = svc.Login(ctx, &model.UserLoginRequest{
		Email:    user.Email,
		Password: "WrongPassword12345!",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	// wrong email
	_, err = svc.Login(ctx, &model.UserLoginRequest{
		Email:    "unknown@example.com",
		Password: "CorrectPassword12345!",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestUserService_RefreshToken(t *testing.T) {
	ctx := context.Background()
	userRepo := repository.NewInMemoryUserRepo()
	rtRepo := repository.NewInMemoryRefreshTokenRepo()
	svc := setupUserServiceForTest(userRepo, rtRepo)

	user := &model.User{ID: 1, Email: "user@example.com", Username: "tester"}
	userRepo.Create(ctx, user)

	validUUID := "d4b4d225-5904-4b6d-badd-d6d4baf81e55"
	expiredUUID := "ab8dac9d-fe96-4efd-9d42-898185465ad6"

	rtRepo.Create(ctx, &model.RefreshToken{
		UserID:    user.ID,
		Value:     uuid.FromStringOrNil(validUUID),
		ExpiresAt: time.Now().Add(time.Hour),
	})
	rtRepo.Create(ctx, &model.RefreshToken{
		UserID:    user.ID,
		Value:     uuid.FromStringOrNil(expiredUUID),
		ExpiresAt: time.Now().Add(-time.Hour),
	})

	// success
	tokenResp, err := svc.RefreshToken(ctx, &model.RefreshTokenRequest{RefreshToken: validUUID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tokenResp.User.Email != user.Email {
		t.Fatalf("expected email %s, got %s", user.Email, tokenResp.User.Email)
	}

	// expired
	_, err = svc.RefreshToken(ctx, &model.RefreshTokenRequest{RefreshToken: expiredUUID})
	if !errors.Is(err, ErrRefreshTokenExpired) {
		t.Fatalf("expected ErrRefreshTokenExpired, got %v", err)
	}

	// invalid
	_, err = svc.RefreshToken(ctx, &model.RefreshTokenRequest{RefreshToken: "not-a-uuid"})
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected ErrInvalidRefreshToken, got %v", err)
	}

	// missing
	_, err = svc.RefreshToken(ctx, &model.RefreshTokenRequest{RefreshToken: "156992d5-b895-4745-9827-ab24787d8492"})
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected ErrInvalidRefreshToken, got %v", err)
	}
}
