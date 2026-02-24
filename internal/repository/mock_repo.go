package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"

	"blog-api/internal/model"
)

// user
type InMemoryUserRepo struct {
	mu    sync.RWMutex
	seq   int
	users map[int]*model.User
}

func NewInMemoryUserRepo() *InMemoryUserRepo {
	return &InMemoryUserRepo{
		users: make(map[int]*model.User),
	}
}

func (r *InMemoryUserRepo) Create(ctx context.Context, user *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.users {
		if u.Email == user.Email || u.Username == user.Username {
			return ErrUserExists
		}
	}

	r.seq++
	user.ID = r.seq
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	r.users[user.ID] = user
	return nil
}

func (r *InMemoryUserRepo) GetByID(ctx context.Context, id int) (*model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}

	copy := *u
	return &copy, nil
}

func (r *InMemoryUserRepo) GetByField(ctx context.Context, field string, value any) (*model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.users {
		switch field {
		case "email":
			if u.Email == value {
				copy := *u
				return &copy, nil
			}
		case "username":
			if u.Username == value {
				copy := *u
				return &copy, nil
			}
		case "id":
			if u.ID == value {
				copy := *u
				return &copy, nil
			}
		default:
			return nil, fmt.Errorf("unsupported field: %s", field)
		}
	}

	return nil, ErrUserNotFound
}

func (r *InMemoryUserRepo) ExistsByField(ctx context.Context, field string, value any) (bool, error) {
	_, err := r.GetByField(ctx, field, value)
	if err != nil {
		if err == ErrUserNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *InMemoryUserRepo) Update(ctx context.Context, user *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.users[user.ID]
	if !ok {
		return ErrUserNotFound
	}

	user.CreatedAt = existing.CreatedAt
	user.UpdatedAt = time.Now()

	r.users[user.ID] = user
	return nil
}

func (r *InMemoryUserRepo) Delete(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[id]; !ok {
		return ErrUserNotFound
	}

	delete(r.users, id)
	return nil
}

func (r *InMemoryUserRepo) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type InMemoryRefreshTokenRepo struct {
	mu     sync.RWMutex
	tokens map[string]*model.RefreshToken
	seq    int
}

func NewInMemoryRefreshTokenRepo() *InMemoryRefreshTokenRepo {
	return &InMemoryRefreshTokenRepo{
		tokens: make(map[string]*model.RefreshToken),
	}
}

func (r *InMemoryRefreshTokenRepo) Store(value string, userID int, expiresAt time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++

	r.tokens[value] = &model.RefreshToken{
		Value:     uuid.FromStringOrNil(value),
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
}

func (r *InMemoryRefreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := token.Value.String()
	if _, exists := r.tokens[key]; exists {
		return errors.New("token already exists")
	}

	r.tokens[key] = token
	return nil
}

func (r *InMemoryRefreshTokenRepo) GetByValue(ctx context.Context, value uuid.UUID) (*model.RefreshToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	token, ok := r.tokens[value.String()]
	if !ok {
		return nil, ErrRefreshTokenNotFound
	}

	copy := *token
	return &copy, nil
}

func (r *InMemoryRefreshTokenRepo) DeleteByValue(ctx context.Context, value uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := value.String()
	if _, ok := r.tokens[key]; !ok {
		return ErrRefreshTokenNotFound
	}

	delete(r.tokens, key)
	return nil
}

func (r *InMemoryRefreshTokenRepo) DeleteByUserID(ctx context.Context, userID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for key, token := range r.tokens {
		if token.UserID == userID {
			delete(r.tokens, key)
		}
	}

	return nil
}

func (r *InMemoryRefreshTokenRepo) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// post
type InMemoryPostRepo struct {
	mu    sync.RWMutex
	seq   int
	posts map[int]*model.Post
}

func NewInMemoryPostRepo() *InMemoryPostRepo {
	return &InMemoryPostRepo{
		posts: make(map[int]*model.Post),
	}
}

func (r *InMemoryPostRepo) Create(ctx context.Context, post *model.Post) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	post.ID = r.seq
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now
	r.posts[post.ID] = post
	return nil
}

func (r *InMemoryPostRepo) GetPost(ctx context.Context, id int, filter *PostFilter) (*model.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	post, ok := r.posts[id]
	if !ok {
		return nil, ErrPostNotFound
	}

	if filter != nil {
		if filter.AuthorID != nil && post.AuthorID != *filter.AuthorID {
			return nil, ErrPostNotFound
		}
		if filter.Published != nil && post.Published != *filter.Published {
			return nil, ErrPostNotFound
		}
		if filter.DueBefore != nil && post.PublishAt != nil && post.PublishAt.After(*filter.DueBefore) {
			return nil, ErrPostNotFound
		}
	}

	copy := *post
	return &copy, nil
}

func (r *InMemoryPostRepo) GetPosts(ctx context.Context, filter *PostFilter, limit, offset int) ([]*model.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*model.Post
	for _, post := range r.posts {
		if filter != nil {
			if filter.AuthorID != nil && post.AuthorID != *filter.AuthorID {
				continue
			}
			if filter.Published != nil && post.Published != *filter.Published {
				continue
			}
			if filter.DueBefore != nil && post.PublishAt != nil && post.PublishAt.After(*filter.DueBefore) {
				continue
			}
		}
		result = append(result, post)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	if offset >= len(result) {
		return []*model.Post{}, nil
	}

	end := offset + limit
	if limit <= 0 || end > len(result) {
		end = len(result)
	}

	copied := make([]*model.Post, end-offset)
	for i := offset; i < end; i++ {
		p := *result[i]
		copied[i-offset] = &p
	}

	return copied, nil
}

func (r *InMemoryPostRepo) GetPostsCount(ctx context.Context, filter *PostFilter) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, post := range r.posts {
		if filter != nil {
			if filter.AuthorID != nil && post.AuthorID != *filter.AuthorID {
				continue
			}
			if filter.Published != nil && post.Published != *filter.Published {
				continue
			}
			if filter.DueBefore != nil && post.PublishAt != nil && post.PublishAt.After(*filter.DueBefore) {
				continue
			}
		}
		count++
	}
	return count, nil
}

func (r *InMemoryPostRepo) Update(ctx context.Context, post *model.Post) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.posts[post.ID]
	if !ok {
		return ErrPostNotFound
	}
	post.CreatedAt = existing.CreatedAt
	post.UpdatedAt = time.Now()
	r.posts[post.ID] = post
	return nil
}

func (r *InMemoryPostRepo) Delete(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.posts[id]; !ok {
		return ErrPostNotFound
	}
	delete(r.posts, id)
	return nil
}

// comment
type InMemoryCommentRepo struct {
	mu       sync.RWMutex
	seq      int
	comments map[int]*model.Comment
}

func NewInMemoryCommentRepo() *InMemoryCommentRepo {
	return &InMemoryCommentRepo{
		comments: make(map[int]*model.Comment),
	}
}

func (r *InMemoryCommentRepo) Create(ctx context.Context, comment *model.Comment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	comment.ID = r.seq
	now := time.Now()
	comment.CreatedAt = now
	comment.UpdatedAt = now
	r.comments[comment.ID] = comment
	return nil
}

func (r *InMemoryCommentRepo) GetByID(ctx context.Context, id int) (*model.Comment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.comments[id]
	if !ok {
		return nil, ErrCommentNotFound
	}
	copy := *c
	return &copy, nil
}

func (r *InMemoryCommentRepo) GetByPostID(ctx context.Context, postID, limit, offset int) ([]*model.Comment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []*model.Comment
	for _, c := range r.comments {
		if c.PostID == postID {
			res = append(res, c)
		}
	}
	if offset >= len(res) {
		return []*model.Comment{}, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(res) {
		end = len(res)
	}
	copied := make([]*model.Comment, end-offset)
	for i := offset; i < end; i++ {
		cc := *res[i]
		copied[i-offset] = &cc
	}
	return copied, nil
}

func (r *InMemoryCommentRepo) GetCountByPostID(ctx context.Context, postID int) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, c := range r.comments {
		if c.PostID == postID {
			count++
		}
	}
	return count, nil
}

func (r *InMemoryCommentRepo) Update(ctx context.Context, comment *model.Comment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.comments[comment.ID]
	if !ok {
		return ErrCommentNotFound
	}
	comment.CreatedAt = existing.CreatedAt
	comment.UpdatedAt = time.Now()
	r.comments[comment.ID] = comment
	return nil
}

func (r *InMemoryCommentRepo) Delete(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.comments[id]; !ok {
		return ErrCommentNotFound
	}
	delete(r.comments, id)
	return nil
}

func (r *InMemoryCommentRepo) GetCountByAuthorID(ctx context.Context, authorID int) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, c := range r.comments {
		if c.AuthorID == authorID {
			count++
		}
	}
	return count, nil
}
