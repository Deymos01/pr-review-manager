package merge

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers"
	"github.com/Deymos01/pr-review-manager/internal/lib/api/response"
	"github.com/Deymos01/pr-review-manager/internal/usecase"
)

type PRService interface {
	MergePullRequest(ctx context.Context, prID string) (*domains.PullRequest, error)
}

type Request struct {
	PrID string `json:"pull_request_id"`
}

type Response struct {
	Pr struct {
		PrID              string    `json:"pull_request_id"`
		PrName            string    `json:"pull_request_name"`
		AuthorID          string    `json:"author_id"`
		Status            string    `json:"status"`
		AssignedReviewers []string  `json:"assigned_reviewers"`
		MergedAt          time.Time `json:"mergedAt"`
	} `json:"pr"`
}

func New(
	log *slog.Logger,
	prService PRService,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http.handlers.pull_requests.merge.New"
		log = log.With(slog.String("op", op))

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Warn("invalid request body", slog.Any("error", err))

			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).
				Encode(response.NewErrorResponse(handlers.InvalidRequest, "invalid JSON format"))
			return
		}

		pr, err := prService.MergePullRequest(r.Context(), req.PrID)
		if err != nil {
			log.Warn("failed to merge pull request", slog.Any("error", err))

			switch {
			case errors.Is(err, usecase.ErrPullRequestNotFound):
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).
					Encode(response.NewErrorResponse(handlers.NotFound, "resource not found"))
			default:
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).
					Encode(response.NewErrorResponse(handlers.InternalError, "internal server error"))
			}
			return
		}

		var resp Response
		resp.Pr.PrID = pr.ID
		resp.Pr.PrName = pr.Name
		resp.Pr.AuthorID = pr.Author.ID
		resp.Pr.Status = pr.Status
		for _, reviewer := range pr.Reviewers {
			resp.Pr.AssignedReviewers = append(resp.Pr.AssignedReviewers, reviewer.User.ID)
		}
		resp.Pr.MergedAt = *pr.MergedAt

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to encode response", slog.Any("error", err))
		}
	}
}
