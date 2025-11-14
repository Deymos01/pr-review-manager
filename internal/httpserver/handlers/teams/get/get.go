package get

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
	GetTeam(ctx context.Context, teamName string) (*domains.Team, error)
}

type MemberResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type Response struct {
	TeamName string           `json:"team_name"`
	Members  []MemberResponse `json:"members"`
}

func New(
	log *slog.Logger,
	service TeamService,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http.handlers.team.get.New"
		log = log.With(slog.String("op", op))

		teamName := r.URL.Query().Get("team_name")
		team, err := service.GetTeam(r.Context(), teamName)
		if err != nil {
			log.Warn("failed to get team", slog.String("team_name", teamName), slog.String("error", err.Error()))

			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).
				Encode(response.NewErrorResponse(handlers.NotFound, "resource not found"))
			return
		}

		var members []MemberResponse
		for _, m := range team.Members {
			members = append(members, MemberResponse{
				UserID:   m.ID,
				Username: m.Name,
				IsActive: m.IsActive,
			})
		}

		resp := Response{
			TeamName: team.Name,
			Members:  members,
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}
	}
}
