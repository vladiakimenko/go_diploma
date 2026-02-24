package service

import (
	"context"
	"errors"
	"testing"

	"blog-api/internal/model"
	"blog-api/internal/repository"
	"blog-api/pkg/logging"
)

func setupCommentServiceForTest() *CommentService {
	postRepo := repository.NewInMemoryPostRepo()
	userRepo := repository.NewInMemoryUserRepo()
	commentRepo := repository.NewInMemoryCommentRepo()

	logging.Init(&logging.LoggerConfig{Level: "DEBUG"})

	return NewCommentService(commentRepo, postRepo, userRepo)
}

func TestCommentServiceCreate(t *testing.T) {
	ctx := context.Background()
	svc := setupCommentServiceForTest()

	userID := 1

	post := &model.Post{
		Title:     "Test Post",
		Content:   "Content",
		Published: true,
		AuthorID:  userID,
	}
	if err := svc.postRepo.Create(ctx, post); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	commentReq := &model.CommentCreateRequest{
		Content: "Hello Comment",
	}
	comment, err := svc.Create(ctx, userID, post.ID, commentReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if comment.Content != commentReq.Content {
		t.Fatalf("expected comment content %q, got %q", commentReq.Content, comment.Content)
	}
}

func TestCommentServiceGetByID(t *testing.T) {
	ctx := context.Background()
	svc := setupCommentServiceForTest()

	userID := 1
	post := &model.Post{
		Title:     "Post",
		Content:   "Content",
		Published: true,
		AuthorID:  userID,
	}
	if err := svc.postRepo.Create(ctx, post); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	comment, _ := svc.Create(ctx, userID, post.ID, &model.CommentCreateRequest{Content: "Test"})

	got, err := svc.GetByID(ctx, comment.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != comment.ID {
		t.Fatalf("expected comment ID %d, got %d", comment.ID, got.ID)
	}

	// missing comment
	_, err = svc.GetByID(ctx, 999)
	if !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got %v", err)
	}
}

func TestCommentServiceGetByPost(t *testing.T) {
	ctx := context.Background()
	svc := setupCommentServiceForTest()

	userID := 1
	post := &model.Post{
		Title:     "Post",
		Content:   "Content",
		Published: true,
		AuthorID:  userID,
	}
	if err := svc.postRepo.Create(ctx, post); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	for i := 0; i < 5; i++ {
		_, _ = svc.Create(ctx, userID, post.ID, &model.CommentCreateRequest{Content: "Comment"})
	}

	pagination := &model.PaginationParams{Limit: ptr(10), Offset: ptr(0)}
	comments, total, err := svc.GetByPost(ctx, post.ID, pagination)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 5 || len(comments) != 5 {
		t.Fatalf("expected 5 comments, got total=%d, len=%d", total, len(comments))
	}
}

func TestCommentServiceUpdate(t *testing.T) {
	ctx := context.Background()
	svc := setupCommentServiceForTest()

	userID := 1
	post := &model.Post{
		Title:     "Post",
		Content:   "Content",
		Published: true,
		AuthorID:  userID,
	}
	if err := svc.postRepo.Create(ctx, post); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	comment, _ := svc.Create(ctx, userID, post.ID, &model.CommentCreateRequest{Content: "Original"})

	// update
	req := &model.CommentUpdateRequest{Content: "Updated"}
	updated, err := svc.Update(ctx, comment.ID, userID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Content != "Updated" {
		t.Fatalf("update did not apply")
	}

	// wrong user
	_, err = svc.Update(ctx, comment.ID, userID+1, req)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	// missing comment
	_, err = svc.Update(ctx, 999, userID, req)
	if !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got %v", err)
	}
}

func TestCommentServiceDelete(t *testing.T) {
	ctx := context.Background()
	svc := setupCommentServiceForTest()

	userID := 1
	post := &model.Post{
		Title:     "Post",
		Content:   "Content",
		Published: true,
		AuthorID:  userID,
	}
	if err := svc.postRepo.Create(ctx, post); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	comment, _ := svc.Create(ctx, userID, post.ID, &model.CommentCreateRequest{Content: "ToDelete"})

	// delete success
	if err := svc.Delete(ctx, comment.ID, userID); err != nil {
		t.Fatalf("expected no error deleting comment, got %v", err)
	}

	// delete missing
	if err := svc.Delete(ctx, comment.ID, userID); !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got %v", err)
	}

	// delete another user's comment
	comment2, _ := svc.Create(ctx, userID, post.ID, &model.CommentCreateRequest{Content: "Other"})
	if err := svc.Delete(ctx, comment2.ID, userID+1); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
