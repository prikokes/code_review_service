package http

import (
	"CodeRewievService/internal/entities"
	"CodeRewievService/internal/interfaces"
	"CodeRewievService/internal/services"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"log/slog"
	"net/http"
)

type TeamHandler struct {
	teamService interfaces.TeamServiceInterface
	logger      *slog.Logger
}

func NewTeamHandler(logger *slog.Logger, db *gorm.DB) *TeamHandler {
	return &TeamHandler{
		logger:      logger,
		teamService: services.NewTeamService(db, logger),
	}
}

func (handler *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var requestBody entities.RequestCreateTeam
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		handler.logger.Error(fmt.Sprintf("Invalid request body: %s", err))
		return
	}

	_, err := handler.teamService.Get(requestBody.TeamName)
	if err == nil {
		handler.logger.Error("Team already exists")

		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(entities.Error{
			Code:    "TEAM_EXISTS",
			Message: "team_name already exists",
		})
		return
	}

	err = handler.teamService.Add(&entities.Team{
		TeamName: requestBody.TeamName,
		Members:  requestBody.Members,
	})

	if err != nil {
		http.Error(w, "Failed to create team", http.StatusBadRequest)
		handler.logger.Error(fmt.Sprintf("Failed to create team: %s", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(entities.ResponseAddTeam{
		Team: entities.Team{
			TeamName: requestBody.TeamName,
			Members:  requestBody.Members,
		},
	})
	if err != nil {
		handler.logger.Error(fmt.Sprintf("Failed to encode team to json: %s", err))
		return
	}
}

func (handler *TeamHandler) MassDeactivateTeamUsers(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		http.Error(w, "Team name is required", http.StatusBadRequest)
		return
	}

	if err := handler.teamService.MassDeactivateTeamUsers(teamName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(map[string]string{
		"message": "Team users deactivated successfully",
		"team":    teamName,
	})
	if err != nil {
		return
	}
}

func (handler *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	handler.logger.Info(fmt.Sprintf("Getting team: %s", teamName))

	team, err := handler.teamService.Get(teamName)

	handler.logger.Info(fmt.Sprintf("Getting team: %s", teamName))

	if err != nil {
		handler.logger.Error(fmt.Sprintf("Team not found: %s", teamName))

		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(entities.Error{
			Code:    "NOT_FOUND",
			Message: "resource not found",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(team)
	if err != nil {
		handler.logger.Error(fmt.Sprintf("Failed to encode team to json: %s", err))
		return
	}
}
