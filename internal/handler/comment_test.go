package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"blog-api/internal/middleware"
	"blog-api/internal/model"
	"blog-api/internal/repository"
	"blog-api/internal/service"
	"blog-api/pkg/logging"
)

// setup
func newCommentTestRouter() http.Handler {
	logging.Init(&logging.LoggerConfig{})

	commentRepo := repository.NewInMemoryCommentRepo()
	userRepo := repository.NewInMemoryUserRepo()
	postRepo := repository.NewInMemoryPostRepo()

	ctx := context.Background()

	userRepo.Create(ctx, &model.User{
		ID:       1,
		Username: "tester",
		Email:    "tester@example.com",
	})

	postRepo.Create(ctx, &model.Post{
		ID:        1,
		Title:     "Test Post",
		Content:   "This is a test post",
		AuthorID:  1,
		Published: true,
	})

	commentRepo.Create(ctx, &model.Comment{
		ID:       1,
		Content:  "Hello",
		PostID:   1,
		AuthorID: 1,
	})

	commentService := service.NewCommentService(commentRepo, postRepo, userRepo)
	commentHandler := NewCommentHandler(commentService)

	router := chi.NewRouter()

	router.Get("/api/posts/{postID}/comments", commentHandler.GetByPost)
	router.Get("/api/posts/{postID}/comments/{commentID}", commentHandler.GetByID)

	router.Group(func(r chi.Router) {
		r.Use(mockAuthMiddleware())

		r.Post(
			"/api/posts/{postID}/comments",
			middleware.ModelBodyMiddleware[model.CommentCreateRequest](commentHandler.Create),
		)

		r.Put(
			"/api/posts/{postID}/comments/{commentID}",
			middleware.ModelBodyMiddleware[model.CommentUpdateRequest](commentHandler.Update),
		)

		r.Delete(
			"/api/posts/{postID}/comments/{commentID}",
			commentHandler.Delete,
		)
	})

	return router
}

// tests
func TestCommentHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		url        string
		body       any
		actorID    int
		wantStatus int
		validateFn func(*testing.T, *http.Response)
	}{
		// create
		{
			name:       "Create comment",
			method:     http.MethodPost,
			url:        "/api/posts/1/comments",
			body:       model.CommentCreateRequest{Content: "Hello"},
			actorID:    1,
			wantStatus: http.StatusCreated,
			validateFn: func(t *testing.T, res *http.Response) {
				validateJsonResponse[model.Comment](t, res)
			},
		},
		{
			name:       "Create comment missing post",
			method:     http.MethodPost,
			url:        "/api/posts/999/comments",
			body:       model.CommentCreateRequest{Content: "Hello"},
			actorID:    1,
			wantStatus: http.StatusNotFound,
			validateFn: nil,
		},

		// read
		{
			name:       "Get comments by post",
			method:     http.MethodGet,
			url:        "/api/posts/1/comments?limit=10&offset=0",
			actorID:    0,
			wantStatus: http.StatusOK,
			validateFn: func(t *testing.T, res *http.Response) {
				validateJsonResponse[model.PaginatedResponse[[]model.Comment]](t, res)
			},
		},
		{
			name:       "Get comment by ID",
			method:     http.MethodGet,
			url:        "/api/posts/1/comments/1",
			actorID:    0,
			wantStatus: http.StatusOK,
			validateFn: func(t *testing.T, res *http.Response) {
				validateJsonResponse[model.Comment](t, res)
			},
		},
		{
			name:       "Get comment by invalid ID",
			method:     http.MethodGet,
			url:        "/api/posts/1/comments/999",
			actorID:    0,
			wantStatus: http.StatusNotFound,
			validateFn: nil,
		},

		// update
		{
			name:       "Update comment owner",
			method:     http.MethodPut,
			url:        "/api/posts/1/comments/1",
			body:       model.CommentUpdateRequest{Content: "Updated"},
			actorID:    1,
			wantStatus: http.StatusOK,
			validateFn: func(t *testing.T, res *http.Response) {
				validateJsonResponse[model.Comment](t, res)
			},
		},
		{
			name:       "Update comment forbidden",
			method:     http.MethodPut,
			url:        "/api/posts/1/comments/1",
			body:       model.CommentUpdateRequest{Content: "Hack"},
			actorID:    2,
			wantStatus: http.StatusForbidden,
			validateFn: nil,
		},
		{
			name:       "Update comment not found",
			method:     http.MethodPut,
			url:        "/api/posts/1/comments/999",
			body:       model.CommentUpdateRequest{Content: "Missing"},
			actorID:    1,
			wantStatus: http.StatusNotFound,
			validateFn: nil,
		},

		// delete
		{
			name:       "Delete comment owner",
			method:     http.MethodDelete,
			url:        "/api/posts/1/comments/1",
			actorID:    1,
			wantStatus: http.StatusNoContent,
			validateFn: nil,
		},
		{
			name:       "Delete comment forbidden",
			method:     http.MethodDelete,
			url:        "/api/posts/1/comments/1",
			actorID:    2,
			wantStatus: http.StatusForbidden,
			validateFn: nil,
		},
		{
			name:       "Delete comment not found",
			method:     http.MethodDelete,
			url:        "/api/posts/1/comments/999",
			actorID:    1,
			wantStatus: http.StatusNotFound,
			validateFn: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset router for each test to avoid shared state
			router := newCommentTestRouter()

			var bodyBytes []byte
			if tt.body != nil {
				var err error
				bodyBytes, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("failed to marshal body: %v", err)
				}
			}

			req := httptest.NewRequest(tt.method, tt.url, bytes.NewReader(bodyBytes))
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			if tt.actorID != 0 {
				req = req.WithContext(setActorID(req.Context(), tt.actorID))
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			res := rec.Result()
			defer res.Body.Close()

			validateStatus(t, res, tt.wantStatus)
			validateHeaders(t, res)

			if tt.validateFn != nil && res.StatusCode < http.StatusBadRequest {
				tt.validateFn(t, res)
			}
		})
	}
}

func TestDelayedPostHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		url        string
		actorID    int
		wantStatus int
		validateFn func(*testing.T, *http.Response)
	}{
		{
			name:       "Get all delayed posts",
			method:     http.MethodGet,
			url:        "/api/delayed?limit=10&offset=0",
			actorID:    1,
			wantStatus: http.StatusOK,
			validateFn: func(t *testing.T, res *http.Response) {
				validateJsonResponse[model.PaginatedResponse[[]model.Post]](t, res)
			},
		},
		{
			name:       "Get delayed post by ID",
			method:     http.MethodGet,
			url:        "/api/delayed/2",
			actorID:    1,
			wantStatus: http.StatusOK,
			validateFn: func(t *testing.T, res *http.Response) {
				validateJsonResponse[model.Post](t, res)
			},
		},
		{
			name:       "Get delayed post by ID forbidden",
			method:     http.MethodGet,
			url:        "/api/delayed/2",
			actorID:    2,
			wantStatus: http.StatusNotFound,
			validateFn: nil,
		},
		{
			name:       "Get delayed post by ID not found",
			method:     http.MethodGet,
			url:        "/api/delayed/999",
			actorID:    1,
			wantStatus: http.StatusNotFound,
			validateFn: nil,
		},
		{
			name:       "Get all delayed posts unauthorized",
			method:     http.MethodGet,
			url:        "/api/delayed?limit=10&offset=0",
			actorID:    0,
			wantStatus: http.StatusUnauthorized,
			validateFn: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newPostTestRouter()

			req := httptest.NewRequest(tt.method, tt.url, nil)
			if tt.actorID != 0 {
				req = req.WithContext(setActorID(req.Context(), tt.actorID))
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			res := rec.Result()
			defer res.Body.Close()

			validateStatus(t, res, tt.wantStatus)
			validateHeaders(t, res)

			if tt.validateFn != nil && res.StatusCode < http.StatusBadRequest {
				tt.validateFn(t, res)
			}
		})
	}
}
