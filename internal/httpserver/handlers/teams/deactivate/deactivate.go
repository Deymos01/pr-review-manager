package deactivate

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

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=TeamService
type TeamService interface {
	DeactivateTeamMembers(ctx context.Context, teamName string, users []string) (*domains.Team, []*domains.ReassignedPR, error)
}

type Request struct {
	TeamName string   `json:"team_name"`
	UserIDs  []string `json:"users"`
}

type Member struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type ReassignedPR struct {
	PrID      string `json:"pull_request_id"`
	OldUserID string `json:"old_reviewer_id"`
	NewUserID string `json:"replaced_by"`
}

type Response struct {
	Team struct {
		Name    string   `json:"team_name"`
		Members []Member `json:"members"`
	} `json:"team"`
	PRs []ReassignedPR `json:"pull_requests"`
}

func New(
	log *slog.Logger,
	service TeamService,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http.handlers.team.deactivate.New"
		log = log.With(slog.String("op", op))

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Warn("invalid request body", slog.Any("error", err))

			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).
				Encode(response.NewErrorResponse(handlers.InvalidRequest, "invalid JSON format"))
			return
		}

		team, reassignedPRs, err := service.DeactivateTeamMembers(r.Context(), req.TeamName, req.UserIDs)
		if err != nil {
			switch {
			case errors.Is(err, usecase.ErrTeamNotFound):
				log.Warn("team not found", slog.String("team_name", req.TeamName))

				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).
					Encode(response.NewErrorResponse(handlers.NotFound, "resource not found"))
			case errors.Is(err, usecase.ErrTeamCompatibility):
				log.Warn("some users do not belong to the team", slog.String("team_name", req.TeamName))

				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).
					Encode(response.NewErrorResponse(handlers.TeamCompatibilityError, "some users do not belong to the team"))
			default:
				log.Error("failed to deactivate members in team",
					slog.String("team_name", team.Name),
					slog.Any("error", err))

				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).
					Encode(response.NewErrorResponse(handlers.InternalError, "failed to deactivate members in team"))
			}
			return
		}

		var resp Response
		resp.Team.Name = team.Name
		resp.Team.Members = make([]Member, 0, len(team.Members))
		resp.PRs = make([]ReassignedPR, 0, len(reassignedPRs))

		for _, m := range team.Members {
			resp.Team.Members = append(resp.Team.Members, Member{
				UserID:   m.ID,
				Username: m.Name,
				IsActive: m.IsActive,
			})
		}

		for _, r := range reassignedPRs {
			resp.PRs = append(resp.PRs, ReassignedPR{
				PrID:      r.PrID,
				OldUserID: r.OldUserID,
				NewUserID: r.NewUserID,
			})
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to encode response", slog.Any("error", err))
		}
	}
}
