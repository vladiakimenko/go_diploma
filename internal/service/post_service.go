package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"blog-api/internal/model"
	"blog-api/internal/repository"
)

const (
	SchedulerInterval        = 1 * time.Minute
	SchedulerWorkers         = 5
	SchedulerBufferSize      = 1000
	SchedulerMaxJobQueueSize = 500
)

// StartPostScheduler launches a background scheduler to publish posts at their scheduled time
func StartPostScheduler(ctx context.Context, repo repository.PostRepository) {
	logger.Info("post scheduler started, interval=%s, workers=%d", SchedulerInterval, SchedulerWorkers)

	ticker := time.NewTicker(SchedulerInterval)
	defer ticker.Stop()

	jobs := make(chan *model.Post, SchedulerBufferSize)
	wg := &sync.WaitGroup{}

	// worker pool
	for i := 0; i < SchedulerWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case post, ok := <-jobs:
					if !ok {
						return
					}
					post.Published = true
					if err := repo.Update(context.Background(), post); err != nil {
						logger.Error("worker %d failed to publish post id=%d: %v", workerID, post.ID, err)
					} else {
						logger.Info("worker %d published post id=%d", workerID, post.ID)
					}
				case <-ctx.Done():
					return
				}
			}
		}(i + 1)
	}

	for {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			logger.Info("post scheduler stopped")
			return
		case <-ticker.C:
			now := time.Now()
			posts, err := repo.GetPosts(
				ctx,
				&repository.PostFilter{
					Published: func(b bool) *bool { return &b }(false),
					DueBefore: &now,
				},
				100, 0,
			)
			if err != nil {
				logger.Error("scheduler failed to fetch due posts: %v", err)
				continue
			}

		PushJobs:
			for _, post := range posts {
				select {
				case jobs <- post:
					if len(jobs) > SchedulerMaxJobQueueSize {
						logger.Warn("post scheduler backlog is large: %d jobs pending", len(jobs))
					}
				case <-ctx.Done():
					break PushJobs
				}
			}
		}
	}
}

var (
	ErrPostNotFound = errors.New("post not found")
)

type PostService struct {
	postRepo repository.PostRepository
	userRepo repository.UserRepository
}

func NewPostService(
	postRepo repository.PostRepository,
	userRepo repository.UserRepository,
) *PostService {
	return &PostService{
		postRepo: postRepo,
		userRepo: userRepo,
	}
}

func (s *PostService) Create(ctx context.Context, userID int, req *model.PostCreateRequest) (*model.Post, error) {
	post := &model.Post{
		Title:     req.Title,
		Content:   req.Content,
		AuthorID:  userID,
		PublishAt: req.PublishAt,
	}

	if post.PublishAt == nil || post.PublishAt.IsZero() || !post.PublishAt.After(time.Now()) {
		post.Published = true
	} else {
		post.Published = false
	}

	if err := s.postRepo.Create(ctx, post); err != nil {
		logger.Error("failed to create post for user_id=%d: %v", userID, err)
		return nil, ErrDatabase
	}

	return post, nil
}

func (s *PostService) GetByID(ctx context.Context, id int) (*model.Post, error) {
	published := true

	post, err := s.postRepo.GetPost(ctx, id, &repository.PostFilter{
		Published: &published,
	})
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			logger.Info("post with id=%d not found or not published", id)
			return nil, ErrPostNotFound
		}

		logger.Error("failed to get post by id=%d: %v", id, err)
		return nil, ErrDatabase
	}

	return post, nil
}

func (s *PostService) GetAll(
	ctx context.Context,
	pagination *model.PaginationParams,
) ([]*model.Post, int, error) {

	published := true
	filter := &repository.PostFilter{
		Published: &published,
	}

	posts, err := s.postRepo.GetPosts(
		ctx,
		filter,
		*pagination.Limit,
		*pagination.Offset,
	)
	if err != nil {
		logger.Error(
			"failed to fetch posts with limit=%d offset=%d: %v",
			*pagination.Limit,
			*pagination.Offset,
			err,
		)
		return nil, 0, ErrDatabase
	}

	total, err := s.postRepo.GetPostsCount(ctx, filter)
	if err != nil {
		logger.Error("failed to count total posts: %v", err)
		return nil, 0, ErrDatabase
	}

	return posts, total, nil
}

func (s *PostService) Update(
	ctx context.Context,
	id int,
	userID int,
	req *model.PostUpdateRequest,
) (*model.Post, error) {

	post, err := s.postRepo.GetPost(ctx, id, nil)
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			logger.Info("post with id=%d not found", id)
			return nil, ErrPostNotFound
		}
		logger.Error("failed to fetch post id=%d: %v", id, err)
		return nil, ErrDatabase
	}

	if err := s.checkPostOwner(post, userID); err != nil {
		return nil, err
	}

	updated := false

	if req.Title != nil && *req.Title != post.Title {
		post.Title = *req.Title
		updated = true
	}

	if req.Content != nil && *req.Content != post.Content {
		post.Content = *req.Content
		updated = true
	}

	if req.PublishAt != nil {
		post.PublishAt = req.PublishAt
		if post.PublishAt != nil && post.PublishAt.After(time.Now()) {
			post.Published = false
		} else {
			post.Published = true
		}
		updated = true
	}

	if !updated {
		return post, nil
	}

	if err := s.postRepo.Update(ctx, post); err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return nil, ErrPostNotFound
		}
		logger.Error("failed to update post id=%d: %v", id, err)
		return nil, ErrDatabase
	}

	return post, nil
}

func (s *PostService) Delete(ctx context.Context, id int, userID int) error {
	post, err := s.postRepo.GetPost(ctx, id, nil)
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			logger.Info("post with id=%d not found", id)
			return ErrPostNotFound
		}
		logger.Error("failed to fetch post id=%d: %v", id, err)
		return ErrDatabase
	}

	if err := s.checkPostOwner(post, userID); err != nil {
		return err
	}

	if err := s.postRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return ErrPostNotFound
		}
		logger.Error("failed to delete post id=%d: %v", id, err)
		return ErrDatabase
	}

	return nil
}

func (s *PostService) GetByAuthor(
	ctx context.Context,
	authorID int,
	pagination *model.PaginationParams,
) ([]*model.Post, int, error) {

	filter := &repository.PostFilter{
		AuthorID:  &authorID,
		Published: func(b bool) *bool { return &b }(true),
	}
	posts, err := s.postRepo.GetPosts(ctx, filter, *pagination.Limit, *pagination.Offset)
	if err != nil {
		logger.Error("failed to fetch posts for author_id=%d: %v", authorID, err)
		return nil, 0, ErrDatabase
	}

	total, err := s.postRepo.GetPostsCount(ctx, filter)
	if err != nil {
		logger.Error("failed to count posts for author_id=%d: %v", authorID, err)
		return nil, 0, ErrDatabase
	}

	return posts, total, nil
}

func (s *PostService) checkPostOwner(post *model.Post, userID int) error {
	if post.AuthorID != userID {
		logger.Info("user_id=%d is not the author of post_id=%d", userID, post.ID)
		return ErrForbidden
	}
	return nil
}

func (s *PostService) GetDelayedPosts(ctx context.Context, userID int, pagination *model.PaginationParams) ([]*model.Post, int, error) {
	filter := &repository.PostFilter{
		AuthorID:  &userID,
		Published: func(b bool) *bool { return &b }(false),
	}

	posts, err := s.postRepo.GetPosts(ctx, filter, *pagination.Limit, *pagination.Offset)
	if err != nil {
		logger.Error("failed to fetch delayed posts for user_id=%d: %v", userID, err)
		return nil, 0, ErrDatabase
	}

	total, err := s.postRepo.GetPostsCount(ctx, filter)
	if err != nil {
		logger.Error("failed to count delayed posts for user_id=%d: %v", userID, err)
		return nil, 0, ErrDatabase
	}

	return posts, total, nil
}

func (s *PostService) GetDelayedPostByID(ctx context.Context, userID, postID int) (*model.Post, error) {
	filter := &repository.PostFilter{
		AuthorID:  &userID,
		Published: func(b bool) *bool { return &b }(false),
	}

	post, err := s.postRepo.GetPost(ctx, postID, filter)
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return nil, ErrPostNotFound
		}
		logger.Error("failed to fetch delayed post id=%d user_id=%d: %v", postID, userID, err)
		return nil, ErrDatabase
	}

	return post, nil
}
