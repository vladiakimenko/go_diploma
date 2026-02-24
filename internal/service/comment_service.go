package service

import (
	"context"
	"errors"

	"blog-api/internal/model"
	"blog-api/internal/repository"
)

var (
	ErrCommentNotFound = errors.New("comment not found")
)

type CommentService struct {
	commentRepo repository.CommentRepository
	postRepo    repository.PostRepository
	userRepo    repository.UserRepository
}

func NewCommentService(
	commentRepo repository.CommentRepository,
	postRepo repository.PostRepository,
	userRepo repository.UserRepository,
) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		postRepo:    postRepo,
		userRepo:    userRepo,
	}
}

func (s *CommentService) Create(
	ctx context.Context,
	userID int,
	postID int,
	req *model.CommentCreateRequest,
) (*model.Comment, error) {

	published := true

	_, err := s.postRepo.GetPost(
		ctx,
		postID, &repository.PostFilter{Published: &published},
	)
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			logger.Info("post with post_id=%d not found or not published", postID)
			return nil, ErrPostNotFound
		}

		logger.Error("failed to fetch post for post_id=%d: %v", postID, err)
		return nil, ErrDatabase
	}

	comment := &model.Comment{
		Content:  req.Content,
		PostID:   postID,
		AuthorID: userID,
	}

	if err := s.commentRepo.Create(ctx, comment); err != nil {
		logger.Error("failed to create comment for post_id=%d, user_id=%d: %v", postID, userID, err)
		return nil, ErrDatabase
	}

	return comment, nil
}

func (s *CommentService) GetByID(ctx context.Context, id int) (*model.Comment, error) {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrCommentNotFound) {
			logger.Info("comment with id=%d not found", id)
			return nil, ErrCommentNotFound
		}
		logger.Error("failed to get comment by id=%d: %v", id, err)
		return nil, ErrDatabase
	}
	return comment, nil
}

func (s *CommentService) GetByPost(
	ctx context.Context,
	postID int,
	pagination *model.PaginationParams,
) ([]*model.Comment, int, error) {

	if err := s.ensurePostPublished(ctx, postID); err != nil {
		return nil, 0, err
	}

	comments, err := s.commentRepo.GetByPostID(ctx, postID, *pagination.Limit, *pagination.Offset)
	if err != nil {
		logger.Error("failed to fetch comments for post_id=%d: %v", postID, err)
		return nil, 0, ErrDatabase
	}

	total, err := s.commentRepo.GetCountByPostID(ctx, postID)
	if err != nil {
		logger.Error("failed to count comments for post_id=%d: %v", postID, err)
		return nil, 0, ErrDatabase
	}

	return comments, total, nil
}

func (s *CommentService) Update(ctx context.Context, id int, userID int, req *model.CommentUpdateRequest) (*model.Comment, error) {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrCommentNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, ErrDatabase
	}

	if err := s.ensurePostPublished(ctx, comment.PostID); err != nil {
		return nil, err
	}

	if err := s.checkOwner(comment, userID); err != nil {
		return nil, err
	}

	comment.Content = req.Content
	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return nil, ErrDatabase
	}
	return comment, nil
}

func (s *CommentService) Delete(ctx context.Context, id int, userID int) error {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrCommentNotFound) {
			logger.Info("comment with id=%d not found", id)
			return ErrCommentNotFound
		}
		logger.Error("failed to get comment by id=%d: %v", id, err)
		return ErrDatabase
	}
	if err := s.checkOwner(comment, userID); err != nil {
		return err
	}
	if err := s.commentRepo.Delete(ctx, id); err != nil {
		logger.Error("failed to delete comment id=%d: %v", id, err)
		return ErrDatabase
	}
	return nil
}

func (s *CommentService) checkOwner(comment *model.Comment, userID int) error {
	if comment.AuthorID != userID {
		logger.Info(
			"user_id=%d is not the author of comment_id=%d (author_id=%d)",
			userID,
			comment.ID,
			comment.AuthorID,
		)
		return ErrForbidden
	}
	return nil
}

func (s *CommentService) ensurePostPublished(ctx context.Context, postID int) error {
	published := true
	_, err := s.postRepo.GetPost(ctx, postID, &repository.PostFilter{Published: &published})
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return ErrCommentNotFound
		}
		logger.Error("failed to fetch post for post_id=%d: %v", postID, err)
		return ErrDatabase
	}
	return nil
}
