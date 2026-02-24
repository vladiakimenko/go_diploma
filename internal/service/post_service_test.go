package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"blog-api/internal/model"
	"blog-api/internal/repository"
	"blog-api/pkg/logging"
)

func ptr[T any](v T) *T { return &v }

func setupPostServiceForTest() *PostService {
	postRepo := repository.NewInMemoryPostRepo()
	userRepo := repository.NewInMemoryUserRepo()

	logging.Init(&logging.LoggerConfig{Level: "DEBUG"})

	return NewPostService(postRepo, userRepo)
}
func TestPostServiceCreate(t *testing.T) {
	ctx := context.Background()
	svc := setupPostServiceForTest()

	userID := 1

	// publish
	req := &model.PostCreateRequest{
		Title:   "Test Post",
		Content: "Hello World",
	}
	post, err := svc.Create(ctx, userID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !post.Published {
		t.Fatalf("expected post to be published immediately")
	}

	// delayed publish
	future := time.Now().Add(time.Hour)
	req2 := &model.PostCreateRequest{
		Title:     "Future Post",
		Content:   "Hello World",
		PublishAt: &future,
	}
	post2, err := svc.Create(ctx, userID, req2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if post2.Published {
		t.Fatalf("expected post to be unpublished (delayed)")
	}
}

func TestPostService_GetByID(t *testing.T) {
	ctx := context.Background()
	svc := setupPostServiceForTest()

	userID := 1
	post, _ := svc.Create(ctx, userID, &model.PostCreateRequest{
		Title:   "Published Post",
		Content: "Hello",
	})

	got, err := svc.GetByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != post.ID {
		t.Fatalf("expected post ID %d, got %d", post.ID, got.ID)
	}

	// missing post
	_, err = svc.GetByID(ctx, 999)
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestPostServiceUpdate(t *testing.T) {
	ctx := context.Background()
	svc := setupPostServiceForTest()

	userID := 1
	post, _ := svc.Create(ctx, userID, &model.PostCreateRequest{
		Title:   "Original",
		Content: "Content",
	})

	newTitle := "Updated"
	newContent := "New Content"
	updated, err := svc.Update(ctx, post.ID, userID, &model.PostUpdateRequest{
		Title:   &newTitle,
		Content: &newContent,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Title != newTitle || updated.Content != newContent {
		t.Fatalf("update did not apply")
	}

	// update non-existent
	_, err = svc.Update(ctx, 999, userID, &model.PostUpdateRequest{
		Title: ptr("X"),
	})
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestPostServiceDelete(t *testing.T) {
	ctx := context.Background()
	svc := setupPostServiceForTest()

	userID := 1
	post, _ := svc.Create(ctx, userID, &model.PostCreateRequest{
		Title:   "ToDelete",
		Content: "Content",
	})

	if err := svc.Delete(ctx, post.ID, userID); err != nil {
		t.Fatalf("expected no error deleting post, got %v", err)
	}

	// delete missing
	if err := svc.Delete(ctx, post.ID, userID); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}

	// delete another user's post
	otherPost, _ := svc.Create(ctx, userID+1, &model.PostCreateRequest{
		Title:   "Other",
		Content: "Content",
	})
	if err := svc.Delete(ctx, otherPost.ID, userID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden deleting another user's post, got %v", err)
	}
}

func TestPostServiceGetAll(t *testing.T) {
	ctx := context.Background()
	svc := setupPostServiceForTest()

	userID := 1
	for i := range 5 {
		svc.Create(ctx, userID, &model.PostCreateRequest{
			Title:   "Post " + string(rune(i)),
			Content: "Content",
		})
	}

	pagination := &model.PaginationParams{Limit: ptr(10), Offset: ptr(0)}
	posts, total, err := svc.GetAll(ctx, pagination)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 5 {
		t.Fatalf("expected total 5, got %d", total)
	}
	if len(posts) != 5 {
		t.Fatalf("expected 5 posts returned, got %d", len(posts))
	}
}

func TestPostServiceGetByAuthor(t *testing.T) {
	ctx := context.Background()
	svc := setupPostServiceForTest()

	userID := 1
	for i := range 5 {
		svc.Create(ctx, userID, &model.PostCreateRequest{
			Title:   "Post " + string(rune(i)),
			Content: "Content",
		})
	}

	pagination := &model.PaginationParams{Limit: ptr(10), Offset: ptr(0)}
	posts, total, err := svc.GetByAuthor(ctx, userID, pagination)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 5 {
		t.Fatalf("expected totalByAuthor 5, got %d", total)
	}
	if len(posts) != 5 {
		t.Fatalf("expected 5 posts returned, got %d", len(posts))
	}
}

func TestPostServiceGetDelayedPosts(t *testing.T) {
	ctx := context.Background()
	svc := setupPostServiceForTest()

	userID := 1
	future := time.Now().Add(time.Hour)
	_, _ = svc.Create(ctx, userID, &model.PostCreateRequest{
		Title:     "Delayed",
		Content:   "Content",
		PublishAt: &future,
	})

	pagination := &model.PaginationParams{Limit: ptr(10), Offset: ptr(0)}
	delayed, total, err := svc.GetDelayedPosts(ctx, userID, pagination)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 || len(delayed) != 1 {
		t.Fatalf("expected 1 delayed post, got %d", total)
	}

	post, err := svc.GetDelayedPostByID(ctx, userID, delayed[0].ID)
	if err != nil {
		t.Fatalf("expected no error fetching delayed post by ID, got %v", err)
	}
	if post.ID != delayed[0].ID {
		t.Fatalf("mismatch delayed post ID")
	}
}
