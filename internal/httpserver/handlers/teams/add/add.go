package add

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers"
	"github.com/Deymos01/pr-review-manager/internal/lib/api/response"
)

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=TeamService
type TeamService interface {
	AddTeam(ctx context.Context, team *domains.Team) (*domains.Team, error)
}

type Member struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type Request struct {
	TeamName string   `json:"team_name"`
	Members  []Member `json:"members"`
}

type Response struct {
	Team struct {
		Name    string   `json:"team_name"`
		Members []Member `json:"members"`
	} `json:"team"`
}

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
				Encode(response.NewErrorResponse(handlers.InvalidRequest, "invalid JSON format"))
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
				Encode(response.NewErrorResponse(handlers.TeamExists, "team_name already exists"))
			return
		}

		var resp Response
		resp.Team.Name = createdTeam.Name

		resp.Team.Members = make([]Member, len(createdTeam.Members))
		for i, m := range createdTeam.Members {
			resp.Team.Members[i] = Member{
				UserID:   m.ID,
				Username: m.Name,
				IsActive: m.IsActive,
			}
		}

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to encode response", slog.Any("error", err))
		}
	}
}
