package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"blog-api/internal/model"
	"blog-api/pkg/database"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

type UserRepo struct {
	db *database.DatabaseManager
}

func NewUserRepo(db *database.DatabaseManager) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	result := r.db.TxDB(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserExists
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int) (*model.User, error) {
	var user model.User
	err := r.db.TxDB(ctx).First(&user, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) GetByField(ctx context.Context, field string, value any) (*model.User, error) {
	var user model.User
	err := r.db.TxDB(ctx).Where(fmt.Sprintf("%s = ?", field), value).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by %s: %w", field, err)
	}
	return &user, nil
}

func (r *UserRepo) ExistsByField(ctx context.Context, field string, value any) (bool, error) {
	var count int64
	err := r.db.TxDB(ctx).Model(&model.User{}).Where(fmt.Sprintf("%s = ?", field), value).Count(&count).Error
	return count > 0, err
}

func (r *UserRepo) Update(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now()
	err := r.db.TxDB(ctx).Save(user).Error
	if err != nil {
		return fmt.Errorf("failed to update user with ID %d: %w", user.ID, err)
	}
	return nil
}

func (r *UserRepo) Delete(ctx context.Context, id int) error {
	result := r.db.TxDB(ctx).Delete(&model.User{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user with ID %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithinTransaction(ctx, fn)
}
