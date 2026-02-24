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
	ErrPostNotFound = errors.New("post not found")
)

// PostFilter defines optional filters for fetching posts.
type PostFilter struct {
	AuthorID  *int
	Published *bool
	DueBefore *time.Time
}

type PostRepo struct {
	db *database.DatabaseManager
}

func NewPostRepo(db *database.DatabaseManager) *PostRepo {
	return &PostRepo{db: db}
}

func (r *PostRepo) Create(ctx context.Context, post *model.Post) error {
	if err := r.db.TxDB(ctx).Create(post).Error; err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}
	return nil
}

func (r *PostRepo) GetPost(
	ctx context.Context,
	id int,
	filter *PostFilter,
) (*model.Post, error) {
	var post model.Post

	db := r.applyFilters(r.db.TxDB(ctx), filter).
		Where("id = ?", id)

	if err := db.First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	return &post, nil
}

func (r *PostRepo) GetPosts(
	ctx context.Context,
	filter *PostFilter,
	limit, offset int,
) ([]*model.Post, error) {
	var posts []*model.Post

	db := r.applyFilters(r.db.TxDB(ctx), filter).
		Order("created_at DESC")

	if limit > 0 {
		db = db.Limit(limit)
	}
	if offset > 0 {
		db = db.Offset(offset)
	}

	if err := db.Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("failed to get posts: %w", err)
	}

	return posts, nil
}

func (r *PostRepo) GetPostsCount(
	ctx context.Context,
	filter *PostFilter,
) (int, error) {
	var count int64

	db := r.applyFilters(r.db.TxDB(ctx), filter).
		Model(&model.Post{})

	if err := db.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count posts: %w", err)
	}

	return int(count), nil
}

func (r *PostRepo) Update(ctx context.Context, post *model.Post) error {
	post.UpdatedAt = time.Now()

	result := r.db.TxDB(ctx).
		Model(&model.Post{}).
		Where("id = ?", post.ID).
		Updates(map[string]any{
			"title":      post.Title,
			"content":    post.Content,
			"publish_at": post.PublishAt,
			"published":  post.Published,
			"updated_at": post.UpdatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update post: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrPostNotFound
	}

	return nil
}

func (r *PostRepo) Delete(ctx context.Context, id int) error {
	result := r.db.TxDB(ctx).Delete(&model.Post{}, id)

	if result.Error != nil {
		return fmt.Errorf("failed to delete post: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrPostNotFound
	}

	return nil
}

func (r *PostRepo) applyFilters(db *gorm.DB, filter *PostFilter) *gorm.DB {
	if filter == nil {
		return db
	}

	if filter.AuthorID != nil {
		db = db.Where("author_id = ?", *filter.AuthorID)
	}

	if filter.Published != nil {
		db = db.Where("published = ?", *filter.Published)
	}

	if filter.DueBefore != nil {
		db = db.Where("publish_at <= ?", *filter.DueBefore)
	}

	return db
}
