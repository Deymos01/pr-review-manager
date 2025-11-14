package set_is_active

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers"
	"github.com/Deymos01/pr-review-manager/internal/lib/api/response"
)

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=UserService
type UserService interface {
	SetUserIsActive(ctx context.Context, userID string, isActive bool) (*domains.User, error)
}

type Request struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type Response struct {
	User struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		TeamName string `json:"team_name"`
		IsActive bool   `json:"is_active"`
	} `json:"user"`
}

func New(
	log *slog.Logger,
	userService UserService,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http.handlers.users.set_is_active.New"
		log = log.With(slog.String("op", op))

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Warn("invalid request body", slog.Any("error", err))
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).
				Encode(response.NewErrorResponse(handlers.InvalidRequest, "invalid JSON format"))
			return
		}

		user, err := userService.SetUserIsActive(r.Context(), req.UserID, req.IsActive)
		if err != nil {
			log.Warn("failed to set user is_active", slog.Any("error", err))
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).
				Encode(response.NewErrorResponse(handlers.NotFound, "resource not found"))
			return
		}

		var resp Response
		resp.User.UserID = user.ID
		resp.User.Username = user.Name
		if user.TeamName != nil {
			resp.User.TeamName = *user.TeamName
		}
		resp.User.IsActive = user.IsActive

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to encode response", slog.Any("error", err))
		}
	}
}
