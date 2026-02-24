package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"

	"blog-api/internal/middleware"
	"blog-api/internal/model"
	"blog-api/internal/repository"
	"blog-api/internal/service"
	"blog-api/pkg/logging"

	"github.com/go-chi/chi/v5"
)

// in-memory repo
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

func (r *InMemoryPostRepo) GetPost(ctx context.Context, id int, filter *repository.PostFilter) (*model.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	post, ok := r.posts[id]
	if !ok {
		return nil, repository.ErrPostNotFound
	}

	if filter != nil {
		if filter.AuthorID != nil && post.AuthorID != *filter.AuthorID {
			return nil, repository.ErrPostNotFound
		}
		if filter.Published != nil && post.Published != *filter.Published {
			return nil, repository.ErrPostNotFound
		}
		if filter.DueBefore != nil && post.PublishAt != nil && post.PublishAt.After(*filter.DueBefore) {
			return nil, repository.ErrPostNotFound
		}
	}

	copy := *post
	return &copy, nil
}

func (r *InMemoryPostRepo) GetPosts(ctx context.Context, filter *repository.PostFilter, limit, offset int) ([]*model.Post, error) {
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

func (r *InMemoryPostRepo) GetPostsCount(ctx context.Context, filter *repository.PostFilter) (int, error) {
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
		return repository.ErrPostNotFound
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
		return repository.ErrPostNotFound
	}
	delete(r.posts, id)
	return nil
}

// utils
func ptr[T any](v T) *T { return &v }

// setup
func newPostTestRouter() http.Handler {
	logging.Init(&logging.LoggerConfig{})

	postRepo := NewInMemoryPostRepo()
	userRepo := NewInMemoryUserRepo()

	ctx := context.Background()
	userRepo.Create(ctx, &model.User{ID: 1, Username: "tester", Email: "tester@example.com"})

	now := time.Now()
	postRepo.Create(ctx, &model.Post{ID: 1, Title: "Test Post", Content: "Content", AuthorID: 1, Published: true, PublishAt: &now})
	postRepo.Create(ctx, &model.Post{ID: 2, Title: "Future Post", Content: "Delayed", AuthorID: 1, Published: false, PublishAt: ptr(now.Add(1 * time.Hour))})

	postService := service.NewPostService(postRepo, userRepo)
	postHandler := NewPostHandler(postService)

	router := chi.NewRouter()

	router.Get("/api/posts", postHandler.GetAll)
	router.Get("/api/posts/{postID}", postHandler.GetByID)

	protected := chi.NewRouter()
	protected.Use(mockAuthMiddleware())

	protected.Post("/api/posts", middleware.ModelBodyMiddleware[model.PostCreateRequest](postHandler.Create))
	protected.Put("/api/posts/{postID}", middleware.ModelBodyMiddleware[model.PostUpdateRequest](postHandler.Update))
	protected.Delete("/api/posts/{postID}", postHandler.Delete)
	
	protected.Get("/api/delayed", postHandler.GetAllDelayed)
	protected.Get("/api/delayed/{postID}", postHandler.GetDelayedByID)

	router.Mount("/", protected)

	return router
}

func TestPostHandler(t *testing.T) {
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
			name:       "Create post",
			method:     http.MethodPost,
			url:        "/api/posts",
			body:       model.PostCreateRequest{Title: "Test Post", Content: "Content", PublishAt: ptr(time.Now())},
			actorID:    1,
			wantStatus: http.StatusCreated,
			validateFn: func(t *testing.T, res *http.Response) { validateJsonResponse[model.Post](t, res) },
		},
		{
			name:       "Create post invalid actor",
			method:     http.MethodPost,
			url:        "/api/posts",
			body:       model.PostCreateRequest{Title: "Test Post", Content: "Content", PublishAt: ptr(time.Now())},
			actorID:    0,
			wantStatus: http.StatusUnauthorized,
			validateFn: nil,
		},
		{
			name:       "Create future post",
			method:     http.MethodPost,
			url:        "/api/posts",
			body:       model.PostCreateRequest{Title: "Future Post", Content: "Delayed", PublishAt: ptr(time.Now().Add(1 * time.Hour))},
			actorID:    1,
			wantStatus: http.StatusCreated,
			validateFn: func(t *testing.T, res *http.Response) { validateJsonResponse[model.Post](t, res) },
		},

		// read
		{
			name:       "Get all posts",
			method:     http.MethodGet,
			url:        "/api/posts?limit=10&offset=0",
			actorID:    0,
			wantStatus: http.StatusOK,
			validateFn: func(t *testing.T, res *http.Response) {
				validateJsonResponse[model.PaginatedResponse[[]model.Post]](t, res)
			},
		},
		{
			name:       "Get post by ID",
			method:     http.MethodGet,
			url:        "/api/posts/1",
			actorID:    0,
			wantStatus: http.StatusOK,
			validateFn: func(t *testing.T, res *http.Response) { validateJsonResponse[model.Post](t, res) },
		},
		{
			name:       "Get post invalid ID",
			method:     http.MethodGet,
			url:        "/api/posts/999",
			actorID:    0,
			wantStatus: http.StatusNotFound,
			validateFn: nil,
		},
		{
			name:       "Get future post by ID (not published)",
			method:     http.MethodGet,
			url:        "/api/posts/2",
			actorID:    0,
			wantStatus: http.StatusNotFound,
			validateFn: nil,
		},

		// update
		{
			name:       "Update post owner",
			method:     http.MethodPut,
			url:        "/api/posts/1",
			body:       model.PostUpdateRequest{Title: ptr("Updated"), Content: ptr("Updated")},
			actorID:    1,
			wantStatus: http.StatusOK,
			validateFn: func(t *testing.T, res *http.Response) { validateJsonResponse[model.Post](t, res) },
		},
		{
			name:       "Update post forbidden",
			method:     http.MethodPut,
			url:        "/api/posts/1",
			body:       model.PostUpdateRequest{Title: ptr("Hack"), Content: ptr("Hack")},
			actorID:    2,
			wantStatus: http.StatusForbidden,
			validateFn: nil,
		},
		{
			name:       "Update post not found",
			method:     http.MethodPut,
			url:        "/api/posts/999",
			body:       model.PostUpdateRequest{Title: ptr("Missing"), Content: ptr("Missing")},
			actorID:    1,
			wantStatus: http.StatusNotFound,
			validateFn: nil,
		},
		{
			name:       "Update future post to publish now",
			method:     http.MethodPut,
			url:        "/api/posts/2",
			body:       model.PostUpdateRequest{PublishAt: ptr(time.Now())},
			actorID:    1,
			wantStatus: http.StatusOK,
			validateFn: func(t *testing.T, res *http.Response) { validateJsonResponse[model.Post](t, res) },
		},

		// delete
		{
			name:       "Delete post owner",
			method:     http.MethodDelete,
			url:        "/api/posts/1",
			actorID:    1,
			wantStatus: http.StatusNoContent,
			validateFn: nil,
		},
		{
			name:       "Delete post forbidden",
			method:     http.MethodDelete,
			url:        "/api/posts/1",
			actorID:    2,
			wantStatus: http.StatusForbidden,
			validateFn: nil,
		},
		{
			name:       "Delete post not found",
			method:     http.MethodDelete,
			url:        "/api/posts/999",
			actorID:    1,
			wantStatus: http.StatusNotFound,
			validateFn: nil,
		},
		{
			name:       "Public endpoint excludes delayed posts",
			method:     http.MethodGet,
			url:        "/api/posts?limit=10&offset=0",
			actorID:    0,
			wantStatus: http.StatusOK,
			validateFn: func(t *testing.T, res *http.Response) {
				var resp model.PaginatedResponse[[]model.Post]
				if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
					t.Fatalf("decode failed: %v", err)
				}
				for _, p := range resp.Data {
					if !p.Published {
						t.Fatalf("found unpublished post in public endpoint: %v", p)
					}
				}
			},
		},

		// delayed
		{
			name:       "Delayed posts unauthenticated",
			method:     http.MethodGet,
			url:        "/api/delayed",
			actorID:    0,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Delayed posts as owner",
			method:     http.MethodGet,
			url:        "/api/delayed",
			actorID:    1,
			wantStatus: http.StatusOK,
			validateFn: func(t *testing.T, res *http.Response) {
				var resp model.PaginatedResponse[[]model.Post]
				if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
					t.Fatalf("decode failed: %v", err)
				}
				if len(resp.Data) != 1 || resp.Data[0].Published {
					t.Fatalf("expected 1 unpublished post, got %v", resp.Data)
				}
			},
		},
		{
			name:       "Delayed post by ID as owner",
			method:     http.MethodGet,
			url:        "/api/delayed/2",
			actorID:    1,
			wantStatus: http.StatusOK,
			validateFn: func(t *testing.T, res *http.Response) {
				var post model.Post
				if err := json.NewDecoder(res.Body).Decode(&post); err != nil {
					t.Fatalf("decode failed: %v", err)
				}
				if post.Published {
					t.Fatalf("expected unpublished post, got published")
				}
			},
		},
		{
			name:       "Delayed post by ID as non-owner",
			method:     http.MethodGet,
			url:        "/api/delayed/2",
			actorID:    2,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newPostTestRouter()

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
