package repository

import (
	"context"

	"github.com/gofrs/uuid/v5"

	"blog-api/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id int) (*model.User, error)
	GetByField(ctx context.Context, field string, value any) (*model.User, error)
	ExistsByField(ctx context.Context, field string, value any) (bool, error)
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	GetByValue(ctx context.Context, value uuid.UUID) (*model.RefreshToken, error)
	DeleteByValue(ctx context.Context, value uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID int) error
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type PostRepository interface {
	Create(ctx context.Context, post *model.Post) error
	GetPost(ctx context.Context, id int, filter *PostFilter) (*model.Post, error)
	GetPosts(ctx context.Context, filter *PostFilter, limit, offset int) ([]*model.Post, error)
	GetPostsCount(ctx context.Context, filter *PostFilter) (int, error)
	Update(ctx context.Context, post *model.Post) error
	Delete(ctx context.Context, id int) error
}

type CommentRepository interface {
	Create(ctx context.Context, comment *model.Comment) error
	GetByID(ctx context.Context, id int) (*model.Comment, error)
	GetByPostID(ctx context.Context, postID int, limit, offset int) ([]*model.Comment, error)
	GetCountByPostID(ctx context.Context, postID int) (int, error)
	GetByAuthorID(ctx context.Context, authorID int, limit, offset int) ([]*model.Comment, error)
	GetCountByAuthorID(ctx context.Context, authorID int) (int, error)
	Update(ctx context.Context, comment *model.Comment) error
	Delete(ctx context.Context, id int) error
}
