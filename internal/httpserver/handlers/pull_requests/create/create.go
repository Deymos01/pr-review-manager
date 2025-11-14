package create

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers"
	"github.com/Deymos01/pr-review-manager/internal/lib/api/response"
	"github.com/Deymos01/pr-review-manager/internal/usecase"
)

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=PRService
type PRService interface {
	CreatePullRequest(ctx context.Context, prID, prName, authorID string) (assignedReviewers []string, err error)
}

type Request struct {
	PrID     string `json:"pull_request_id"`
	PrName   string `json:"pull_request_name"`
	AuthorID string `json:"author_id"`
}

type Response struct {
	PR struct {
		PrID              string   `json:"pull_request_id"`
		PrName            string   `json:"pull_request_name"`
		AuthorID          string   `json:"author_id"`
		Status            string   `json:"status"`
		AssignedReviewers []string `json:"assigned_reviewers"`
	} `json:"pr"`
}

func New(
	log *slog.Logger,
	prService PRService,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http.handlers.pull_requests.create.New"
		log = log.With(slog.String("op", op))

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Warn("invalid request body", slog.Any("error", err))

			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).
				Encode(response.NewErrorResponse(handlers.InvalidRequest, "invalid JSON format"))
			return
		}

		assignedReviewers, err := prService.CreatePullRequest(r.Context(), req.PrID, req.PrName, req.AuthorID)
		if err != nil {
			log.Warn("failed to create pull request", slog.Any("error", err))

			switch {
			case errors.Is(err, usecase.ErrUserNotFound) || errors.Is(err, usecase.ErrTeamNotFound):
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).
					Encode(response.NewErrorResponse(handlers.NotFound, "resource not found"))
			case errors.Is(err, usecase.ErrPRAlreadyExists):
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).
					Encode(response.NewErrorResponse(handlers.PrExists, "pull request already exists"))
			default:
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).
					Encode(response.NewErrorResponse(handlers.InternalError, "internal server error"))
			}

			return
		}

		var resp Response
		resp.PR.PrID = req.PrID
		resp.PR.PrName = req.PrName
		resp.PR.AuthorID = req.AuthorID
		resp.PR.Status = "OPEN"
		resp.PR.AssignedReviewers = assignedReviewers

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to encode response", slog.Any("error", err))
		}
	}
}
