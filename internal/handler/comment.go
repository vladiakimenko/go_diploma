package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"blog-api/internal/model"
	"blog-api/internal/service"
	"blog-api/pkg/exception"
)

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
	}
}

// GET /api/posts/{postID}/comments?limit=20&offset=0
func (h *CommentHandler) GetByPost(w http.ResponseWriter, r *http.Request) {

	pagination, ok := getPaginationParams(r)
	if !ok {
		exception.WriteApiError(w, exception.BadRequestError("Invalid pagination parameters"))
		return
	}

	postIDStr := chi.URLParam(r, "postID")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		exception.WriteApiError(w, exception.BadRequestError("Invalid post ID"))
		return
	}

	result, total, err := h.commentService.GetByPost(r.Context(), postID, pagination)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	writePaginatedJSON(w, http.StatusOK, result, pagination, total)
}

// POST /api/posts/{postID}/comments
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	body, ok := getParsedBody[model.CommentCreateRequest](r)
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

	result, err := h.commentService.Create(r.Context(), actorID, postID, body)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// GET /api/posts/{postID}/comments/{commentID}
func (h *CommentHandler) GetByID(w http.ResponseWriter, r *http.Request) {

	commentIDStr := chi.URLParam(r, "commentID")
	commentID, err := strconv.Atoi(commentIDStr)
	if err != nil {
		exception.WriteApiError(w, exception.BadRequestError("Invalid comment ID"))
		return
	}

	result, err := h.commentService.GetByID(r.Context(), commentID)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// PUT /api/posts/{postID}/comments/{commentID}
func (h *CommentHandler) Update(w http.ResponseWriter, r *http.Request) {

	body, ok := getParsedBody[model.CommentUpdateRequest](r)
	if !ok {
		exception.WriteApiError(w, exception.BadRequestError("Invalid request body"))
		return
	}

	actorID, ok := getActorID(r.Context())
	if !ok {
		exception.WriteApiError(w, exception.InternalServerError("Auth misconfigured"))
		return
	}

	commentIDStr := chi.URLParam(r, "commentID")
	commentID, err := strconv.Atoi(commentIDStr)
	if err != nil {
		exception.WriteApiError(w, exception.BadRequestError("Invalid comment ID"))
		return
	}

	result, err := h.commentService.Update(r.Context(), commentID, actorID, body)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// DELETE /api/posts/{postID}/comments/{commentID}
func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actorID, ok := getActorID(r.Context())
	if !ok {
		exception.WriteApiError(w, exception.InternalServerError("Auth misconfigured"))
		return
	}

	commentIDStr := chi.URLParam(r, "commentID")
	commentID, err := strconv.Atoi(commentIDStr)
	if err != nil {
		exception.WriteApiError(w, exception.BadRequestError("Invalid comment ID"))
		return
	}

	err = h.commentService.Delete(r.Context(), commentID, actorID)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
