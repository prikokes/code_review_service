package http

import (
	"CodeRewievService/internal/entities"
	"CodeRewievService/internal/interfaces"
	"CodeRewievService/internal/services"
	"encoding/json"
	"gorm.io/gorm"
	"log/slog"
	"net/http"
)

type StatsHandler struct {
	logger       *slog.Logger
	statsService interfaces.StatsServiceInterface
}

func NewStatsHandler(logger *slog.Logger, db *gorm.DB) *StatsHandler {
	return &StatsHandler{
		logger:       logger,
		statsService: services.NewStatsService(db),
	}
}

func (handler *StatsHandler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("teamName")
	if teamName == "" {
		handler.writeError(w, "team_name is required", http.StatusBadRequest)
		return
	}

	teamExists, err := handler.statsService.TeamExists(teamName)
	if err != nil {
		handler.logger.Error("failed to check team existence", "error", err, "team", teamName)
		handler.writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !teamExists {
		handler.writeError(w, "team not found", http.StatusNotFound)
		return
	}

	userStats, err := handler.statsService.GetUserStats(teamName)
	if err != nil {
		handler.logger.Error("failed to get user stats", "error", err, "team", teamName)
		handler.writeError(w, "failed to get user statistics", http.StatusInternalServerError)
		return
	}

	handler.writeJSON(w, userStats, http.StatusOK)
}

func (handler *StatsHandler) GetTeamStats(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		handler.writeError(w, "team_name is required", http.StatusBadRequest)
		return
	}

	teamExists, err := handler.statsService.TeamExists(teamName)
	if err != nil {
		handler.logger.Error("failed to check team existence", "error", err, "team", teamName)
		handler.writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !teamExists {
		handler.writeError(w, "team not found", http.StatusNotFound)
		return
	}

	teamStats, err := handler.statsService.GetTeamStats(teamName)
	if err != nil {
		handler.logger.Error("failed to get team stats", "error", err, "team", teamName)
		handler.writeError(w, "failed to get team statistics", http.StatusInternalServerError)
		return
	}

	handler.writeJSON(w, teamStats, http.StatusOK)
}

func (handler *StatsHandler) writeJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		handler.logger.Error("failed to encode JSON response", "error", err)
	}
}

func (handler *StatsHandler) writeError(w http.ResponseWriter, message string, statusCode int) {
	handler.writeJSON(w, entities.ErrorStatsResponse{Error: message}, statusCode)
}
