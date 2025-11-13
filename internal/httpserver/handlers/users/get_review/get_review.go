package get_review

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers"
	"github.com/Deymos01/pr-review-manager/internal/lib/api/response"
)

type UserService interface {
	GetUsersReview(ctx context.Context, userID string) ([]*domains.PullRequest, error)
}

type Response struct {
	UserID string `json:"user_id"`
	PRs    struct {
		PrID     string `json:"pull_request_id"`
		PrName   string `json:"pull_request_name"`
		AuthorID string `json:"author_id"`
		Status   string `json:"status"`
	} `json:"pull_requests"`
}

func New(
	log *slog.Logger,
	userService UserService,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http.handlers.users.get_review.New"
		log = log.With(slog.String("op", op))

		userID := r.URL.Query().Get("user_id")

		reviews, err := userService.GetUsersReview(r.Context(), userID)
		if err != nil {
			log.Warn("failed to get user reviews", slog.Any("error", err))
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).
				Encode(response.NewErrorResponse(handlers.NotFound, "resource not found"))
			return
		}

		var resp Response
		resp.UserID = userID
		for _, pr := range reviews {
			resp.PRs.PrID = pr.ID
			resp.PRs.PrName = pr.Name
			resp.PRs.AuthorID = pr.Author.ID
			resp.PRs.Status = pr.Status
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to encode response", slog.Any("error", err))
		}
	}
}
