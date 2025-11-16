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

type UserHandler struct {
	userService interfaces.UserServiceInterface
	logger      *slog.Logger
}

func NewUserHandler(logger *slog.Logger, db *gorm.DB) *UserHandler {
	return &UserHandler{
		userService: services.NewUserService(db),
		logger:      logger,
	}
}

func (handler *UserHandler) SetUserIsActive(w http.ResponseWriter, r *http.Request) {
	var requestBody entities.RequestSetIsActive
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		handler.logger.Error(fmt.Sprintf("Invalid request body: %s", err))
		return
	}

	newUser, err := handler.userService.SetIsActive(&entities.User{
		UserID:   requestBody.UserID,
		IsActive: requestBody.IsActive,
	})

	if err != nil {
		handler.logger.Error(fmt.Sprintf("Error setting user isActive: %s", err))

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
	err = json.NewEncoder(w).Encode(entities.ResponseSetIsActive{
		User: *newUser,
	})
	if err != nil {
		handler.logger.Error(fmt.Sprintf("Error encoding response: %s", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (handler *UserHandler) GetUserReview(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")

	userReview, err := handler.userService.GetReview(userID)

	if err != nil {
		handler.logger.Error(fmt.Sprintf("Error getting user review: %s", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(userReview)
	if err != nil {
		handler.logger.Error(fmt.Sprintf("Error encoding response: %s", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
