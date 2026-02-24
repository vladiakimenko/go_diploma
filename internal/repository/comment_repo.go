package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"blog-api/internal/model"
	"blog-api/pkg/database"
)

var (
	ErrCommentNotFound = errors.New("comment not found")
)

type CommentRepo struct {
	db *database.DatabaseManager
}

func NewCommentRepo(db *database.DatabaseManager) *CommentRepo {
	return &CommentRepo{db: db}
}

func (r *CommentRepo) Create(ctx context.Context, comment *model.Comment) error {
	err := r.db.TxDB(ctx).Create(comment).Error
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}
	return nil
}

func (r *CommentRepo) GetByID(ctx context.Context, id int) (*model.Comment, error) {
	var comment model.Comment
	err := r.db.TxDB(ctx).First(&comment, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("failed to get comment by ID: %w", err)
	}
	return &comment, nil
}

func (r *CommentRepo) GetByPostID(ctx context.Context, postID int, limit, offset int) ([]*model.Comment, error) {
	var comments []*model.Comment
	err := r.db.TxDB(ctx).Where("post_id = ?", postID).Order("created_at ASC").Limit(limit).Offset(offset).Find(&comments).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get comments for post %d: %w", postID, err)
	}
	return comments, nil
}

func (r *CommentRepo) GetCountByPostID(ctx context.Context, postID int) (int, error) {
	var count int64
	err := r.db.TxDB(ctx).Model(&model.Comment{}).Where("post_id = ?", postID).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count comments for post %d: %w", postID, err)
	}
	return int(count), nil
}

func (r *CommentRepo) GetCountByAuthorID(ctx context.Context, authorID int) (int, error) {
	var count int64
	err := r.db.TxDB(ctx).
		Model(&model.Comment{}).
		Where("author_id = ?", authorID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count comments by author ID %d: %w", authorID, err)
	}
	return int(count), nil
}

func (r *CommentRepo) Update(ctx context.Context, comment *model.Comment) error {
	comment.UpdatedAt = time.Now()

	result := r.db.TxDB(ctx).Model(&model.Comment{}).Where("id = ?", comment.ID).Updates(
		map[string]interface{}{
			"content":    comment.Content,
			"updated_at": comment.UpdatedAt,
		},
	)
	if result.Error != nil {
		return fmt.Errorf("failed to update comment: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrCommentNotFound
	}
	return nil
}

func (r *CommentRepo) Delete(ctx context.Context, id int) error {
	result := r.db.TxDB(ctx).Delete(&model.Comment{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete comment: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrCommentNotFound
	}
	return nil
}
