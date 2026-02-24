package handler

import (
	"blog-api/internal/model"
	"blog-api/internal/service"
	"blog-api/pkg/exception"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type PostHandler struct {
	postService *service.PostService
}

func NewPostHandler(postService *service.PostService) *PostHandler {
	return &PostHandler{
		postService: postService,
	}
}

// GET /api/posts?limit=10&offset=0&author={authorID}
func (h *PostHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	pagination, ok := getPaginationParams(r)
	if !ok {
		exception.WriteApiError(w, exception.BadRequestError("Invalid pagination parameters"))
		return
	}

	authorIDStr := r.URL.Query().Get("author")
	var (
		result []*model.Post
		total  int
		err    error
	)

	if authorIDStr != "" {
		authorID, err := strconv.Atoi(authorIDStr)
		if err != nil {
			exception.WriteApiError(w, exception.BadRequestError("Invalid author ID"))
			return
		}
		result, total, err = h.postService.GetByAuthor(r.Context(), authorID, pagination)
	} else {
		result, total, err = h.postService.GetAll(r.Context(), pagination)
	}

	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	writePaginatedJSON(w, http.StatusOK, result, pagination, total)
}

// GET /api/posts/{postID}
func (h *PostHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "postID")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		exception.WriteApiError(w, exception.BadRequestError("Invalid post ID"))
		return
	}

	result, err := h.postService.GetByID(r.Context(), postID)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// POST /api/posts
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	body, ok := getParsedBody[model.PostCreateRequest](r)
	if !ok {
		exception.WriteApiError(w, exception.BadRequestError("Invalid request body"))
		return
	}

	actorID, ok := getActorID(r.Context())
	if !ok {
		exception.WriteApiError(w, exception.InternalServerError("Auth misconfigured"))
		return
	}

	result, err := h.postService.Create(r.Context(), actorID, body)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// PUT /api/posts/{postID}
func (h *PostHandler) Update(w http.ResponseWriter, r *http.Request) {
	body, ok := getParsedBody[model.PostUpdateRequest](r)
	if !ok {
		exception.WriteApiError(w, exception.BadRequestError("Invalid request body"))
		return
	}

	actorID, ok := getActorID(r.Context())
	if !ok {
		exception.WriteApiError(w, exception.InternalServerError("Auth misconfigured"))
		return
	}

	postIDStr := chi.URLParam(r, "postID")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		exception.WriteApiError(w, exception.BadRequestError("Invalid post ID"))
		return
	}

	result, err := h.postService.Update(r.Context(), postID, actorID, body)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// DELETE /api/posts/{postID}
func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actorID, ok := getActorID(r.Context())
	if !ok {
		exception.WriteApiError(w, exception.InternalServerError("Auth misconfigured"))
		return
	}

	postIDStr := chi.URLParam(r, "postID")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		exception.WriteApiError(w, exception.BadRequestError("Invalid post ID"))
		return
	}

	err = h.postService.Delete(r.Context(), postID, actorID)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// separate auth required endpoints for fetching own delayed posts (write-actions are realized via '/api/posts/{postID}')
// GET /api/delayed
func (h *PostHandler) GetAllDelayed(w http.ResponseWriter, r *http.Request) {
	pagination, ok := getPaginationParams(r)
	if !ok {
		exception.WriteApiError(w, exception.BadRequestError("Invalid pagination parameters"))
		return
	}

	userID, ok := getActorID(r.Context())
	if !ok {
		exception.WriteApiError(w, exception.UnauthorizedError("Missing authentication"))
		return
	}

	posts, total, err := h.postService.GetDelayedPosts(r.Context(), userID, pagination)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	writePaginatedJSON(w, http.StatusOK, posts, pagination, total)
}

// GET /api/delayed
func (h *PostHandler) GetDelayedByID(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "postID")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		exception.WriteApiError(w, exception.BadRequestError("Invalid post ID"))
		return
	}

	userID, ok := getActorID(r.Context())
	if !ok {
		exception.WriteApiError(w, exception.UnauthorizedError("Missing authentication"))
		return
	}

	post, err := h.postService.GetDelayedPostByID(r.Context(), userID, postID)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	writeJSON(w, http.StatusOK, post)
}
