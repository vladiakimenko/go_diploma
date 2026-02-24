package model

import (
	"errors"
	"regexp"
	"time"

	"github.com/gofrs/uuid/v5"
)

// domain
type User struct {
	ID           int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Username     string    `json:"username" gorm:"unique;not null"`
	Email        string    `json:"email" gorm:"unique;not null"`
	Password     string    `json:"password,omitempty" gorm:"-"`
	PasswordHash string    `json:"-" gorm:"column:password_hash;not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type RefreshToken struct {
	Value     uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    int       `gorm:"index;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time
}

type Post struct {
	ID        int        `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	Title     string     `json:"title" db:"title" gorm:"not null"`
	Content   string     `json:"content" db:"content" gorm:"not null"`
	AuthorID  int        `json:"author_id" db:"author_id" gorm:"not null;index"`
	CreatedAt time.Time  `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`
	PublishAt *time.Time `json:"publish_at,omitempty" db:"publish_at" gorm:"column:publish_at"`
	Published bool       `json:"published" db:"published" gorm:"default:false"`
}

type Comment struct {
	ID        int       `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	Content   string    `json:"content" db:"content" gorm:"not null"`
	PostID    int       `json:"post_id" db:"post_id" gorm:"not null;index"`
	AuthorID  int       `json:"author_id" db:"author_id" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`
}

// requests
type UserCreateRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

func (r *UserCreateRequest) CustomValidate() error {
	emailRegex := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	matched, err := regexp.MatchString(emailRegex, r.Email)
	if err != nil {
		return err
	}
	if !matched {
		return errors.New("invalid email format")
	}
	/* 	NOTE: Password validation must be ensured with
	auth.PasswordManager.ValidatePasswordStrength at the service layer
	(since service.UserService already owns an instance) */
	return nil
}

type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required,uuid4"`
}

type PostCreateRequest struct {
	Title     string     `json:"title" validate:"required,min=1,max=200"`
	Content   string     `json:"content" validate:"required,min=1"`
	PublishAt *time.Time `json:"publish_at,omitempty"`
}

type PostUpdateRequest struct {
	Title     *string    `json:"title,omitempty" validate:"omitempty,max=200"`
	Content   *string    `json:"content,omitempty" validate:"omitempty,min=1"`
	PublishAt *time.Time `json:"publish_at,omitempty"`
}

type CommentCreateRequest struct {
	Content string `json:"content" validate:"required,min=1,max=1000"`
}

type CommentUpdateRequest struct {
	Content string `json:"content" validate:"required,min=1,max=1000"`
}

// GET params
type PaginationParams struct {
	Limit  *int `form:"limit" validate:"omitempty,min=0,max=100"`
	Offset *int `form:"offset" validate:"omitempty,min=0"`
}

func (p *PaginationParams) SetDefaults() {
	if p.Limit == nil {
		defaultLimit := 20
		p.Limit = &defaultLimit
	}
	if p.Offset == nil {
		defaultOffset := 0
		p.Offset = &defaultOffset
	}
}

// responses
type TokenResponse struct {
	AccessToken        string    `json:"access_token"`
	AccessTokenExpiry  time.Time `json:"access_token_expires_at"`
	RefreshToken       string    `json:"refresh_token"`
	RefreshTokenExpiry time.Time `json:"refresh_token_expires_at"`
	User               *User     `json:"user"`
}

type PaginatedResponse[T any] struct {
	Data   T   `json:"data"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}
