package repository

import (
	"context"
	"errors"

	"blog-api/internal/model"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"

	"blog-api/pkg/database"
)

var ErrRefreshTokenNotFound = errors.New("refresh token not found")

type RefreshTokenRepo struct {
	db *database.DatabaseManager
}

func NewRefreshTokenRepo(db *database.DatabaseManager) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

func (r *RefreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	return r.db.TxDB(ctx).Create(token).Error
}

func (r *RefreshTokenRepo) GetByValue(
	ctx context.Context,
	value uuid.UUID,
) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.db.TxDB(ctx).
		First(&rt, "value = ?", value).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, err
	}
	return &rt, nil
}

func (r *RefreshTokenRepo) DeleteByValue(
	ctx context.Context,
	value uuid.UUID,
) error {
	res := r.db.TxDB(ctx).
		Delete(&model.RefreshToken{}, "value = ?", value)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrRefreshTokenNotFound
	}
	return nil
}

func (r *RefreshTokenRepo) DeleteByUserID(
	ctx context.Context,
	userID int,
) error {
	return r.db.TxDB(ctx).
		Where("user_id = ?", userID).
		Delete(&model.RefreshToken{}).Error
}

func (r *RefreshTokenRepo) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithinTransaction(ctx, fn)
}
