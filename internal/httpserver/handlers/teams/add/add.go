package add

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/lib/api/response"
)

type TeamService interface {
	AddTeam(ctx context.Context, team *domains.Team) (*domains.Team, error)
}

type MemberRequest struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type Request struct {
	TeamName string          `json:"team_name"`
	Members  []MemberRequest `json:"members"`
}

type Response struct {
	Team domains.Team `json:"team"`
}

const (
	invalidRequest = "INVALID_REQUEST"
	teamExists     = "TEAM_EXISTS"
)

func New(
	log *slog.Logger,
	service TeamService,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http.handlers.team.add.New"
		log = log.With(slog.String("op", op))

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Warn("invalid request body", slog.Any("error", err))

			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).
				Encode(response.NewErrorResponse(invalidRequest, "invalid JSON format"))
			return
		}

		var members []*domains.User
		for _, m := range req.Members {
			members = append(members, &domains.User{
				ID:       m.UserID,
				Name:     m.Username,
				TeamName: &req.TeamName,
				IsActive: m.IsActive,
			})
		}
		team := domains.Team{
			Name:    req.TeamName,
			Members: members,
		}

		createdTeam, err := service.AddTeam(r.Context(), &team)
		if err != nil {
			log.Error("failed to create team", slog.Any("error", err))

			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).
				Encode(response.NewErrorResponse(teamExists, "team_name already exists"))
			return
		}

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(Response{Team: *createdTeam}); err != nil {
			log.Error("failed to encode response", slog.Any("error", err))
		}
	}
}
