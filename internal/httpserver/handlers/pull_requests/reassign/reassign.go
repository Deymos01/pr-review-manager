package reassign

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers"
	"github.com/Deymos01/pr-review-manager/internal/lib/api/response"
	"github.com/Deymos01/pr-review-manager/internal/usecase"
)

type PRService interface {
	ReassignReviewer(ctx context.Context, prID, oldUserID string) (*domains.PullRequest, string, error)
}

type Request struct {
	PrID      string `json:"pull_request_id"`
	OldUserID string `json:"old_reviewer_id"`
}

type Response struct {
	Pr struct {
		PrID              string   `json:"pull_request_id"`
		PrName            string   `json:"pull_request_name"`
		AuthorID          string   `json:"author_id"`
		Status            string   `json:"status"`
		AssignedReviewers []string `json:"assigned_reviewers"`
	} `json:"pr"`
	NewUserID string `json:"replaced_by"`
}

func New(
	log *slog.Logger,
	prService PRService,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http.handlers.pull_requests.reassign.New"
		log = log.With(slog.String("op", op))

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Warn("invalid request body", slog.Any("error", err))

			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).
				Encode(response.NewErrorResponse(handlers.InvalidRequest, "invalid JSON format"))
			return
		}

		pr, newUserID, err := prService.ReassignReviewer(r.Context(), req.PrID, req.OldUserID)
		if err != nil {
			log.Warn("failed to reassign reviewer", slog.Any("error", err))

			switch {
			case errors.Is(err, usecase.ErrPullRequestNotFound) || errors.Is(err, usecase.ErrUserNotFound):
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).
					Encode(response.NewErrorResponse(handlers.NotFound, "resource not found"))
			case errors.Is(err, usecase.ErrPRAlreadyMerged):
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).
					Encode(response.NewErrorResponse(handlers.PrMerged, "cannot reassign on merged PR"))
			case errors.Is(err, usecase.ErrUserNotAssigned):
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).
					Encode(response.NewErrorResponse(handlers.NotAssigned, "reviewer is not assigned to this PR"))
			case errors.Is(err, usecase.ErrNoAvailableReviewer):
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).
					Encode(response.NewErrorResponse(handlers.NoCandidate, "no active replacement candidate in team"))
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
		resp.NewUserID = newUserID

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to encode response", slog.Any("error", err))
		}
	}
}
