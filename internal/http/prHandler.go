package http

import (
	"CodeRewievService/internal/entities"
	"CodeRewievService/internal/interfaces"
	"CodeRewievService/internal/services"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"log/slog"
	"net/http"
	"time"
)

type PrHandler struct {
	prService interfaces.PullRequestServiceInterface
	logger    *slog.Logger
}

func NewPrHandler(logger *slog.Logger, db *gorm.DB) *PrHandler {
	return &PrHandler{
		prService: services.NewPullRequestService(db),
		logger:    logger,
	}
}

func (handler *PrHandler) CreatePR(w http.ResponseWriter, r *http.Request) {
	var requestBody entities.RequestCreatePR
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		handler.logger.Error(fmt.Sprintf("Invalid request body: %s", err))
		return
	}

	pr, err := handler.prService.Create(&entities.PullRequest{
		PullRequestID:   requestBody.PullRequestID,
		PullRequestName: requestBody.PullRequestName,
		AuthorID:        requestBody.AuthorID,
		Status:          "OPEN",
	})

	if errors.Is(err, entities.ErrPRAlreadyExists) {
		handler.logger.Error(fmt.Sprintf("PR already exists: %s", requestBody.PullRequestID))

		w.WriteHeader(http.StatusConflict)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(entities.Error{
			Code:    "PR_EXISTS",
			Message: "PR id already exists",
		})
		return
	}

	if errors.Is(err, entities.ErrAuthorNotFound) {
		handler.logger.Error(fmt.Sprintf("Author / team not found: %s", requestBody.AuthorID))

		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(entities.Error{
			Code:    "NOT_FOUND",
			Message: "resource not found",
		})
		return
	}

	if err != nil {
		handler.logger.Error(fmt.Sprintf("PR creation error: %s", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(entities.ResponseCreatePR{
		PullRequest: pr.ToResponse(),
	})
	if err != nil {
		handler.logger.Error(fmt.Sprintf("Failed to encode pr to json: %s", err))
		return
	}
}

func (handler *PrHandler) MergePR(w http.ResponseWriter, r *http.Request) {
	var requestBody entities.RequestMergePR
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		handler.logger.Error(fmt.Sprintf("Invalid request body: %s", err))
		return
	}

	pr, err := handler.prService.Merge(requestBody.PullRequestID)

	if errors.Is(err, entities.ErrNotFound) {
		handler.logger.Error(fmt.Sprint("Author / team not found"))

		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(entities.Error{
			Code:    "NOT_FOUND",
			Message: "resource not found",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(entities.ResponseMerge{
		PullRequest: pr.ToResponse(),
		MergedAT:    time.Now(),
	})
	if err != nil {
		handler.logger.Error(fmt.Sprintf("Failed to encode pr to json: %s", err))
		return
	}
}

func (handler *PrHandler) ReassignPR(w http.ResponseWriter, r *http.Request) {
	var requestBody entities.RequestReassignPR
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		handler.logger.Error(fmt.Sprintf("Invalid request body: %s", err))
		return
	}

	pr, userId, err := handler.prService.Reassign(requestBody.PullRequestID, requestBody.OldReviewerID)

	if errors.Is(err, entities.ErrPRAlreadyMerged) {
		handler.logger.Error(fmt.Sprintf("User not assigned to this pr: %s", requestBody.PullRequestID))

		w.WriteHeader(http.StatusConflict)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(entities.Error{
			Code:    "PR_MERGED",
			Message: "cannot reassign on merged PR",
		})
		return
	}

	if errors.Is(err, entities.ErrUserIsNotAssignedToPR) {
		handler.logger.Error(fmt.Sprintf("Reviewer is not assigned to this PR: pr=%s",
			requestBody.PullRequestID))

		w.WriteHeader(http.StatusConflict)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(entities.Error{
			Code:    "NOT_ASSIGNED",
			Message: "reviewer not assigned to this PR",
		})
		return
	}

	if errors.Is(err, entities.ErrNotFound) {
		handler.logger.Error(fmt.Sprintf("PR / author is not found: pr=%s", requestBody.PullRequestID))

		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(entities.Error{
			Code:    "NOT_FOUND",
			Message: "resource not found",
		})
		return
	}

	if errors.Is(err, entities.ErrNoReplacement) {
		handler.logger.Error(fmt.Sprint("No replacement found for reviewer"))

		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(entities.Error{
			Code:    "NOT_FOUND",
			Message: "resource not found",
		})
		return
	}

	if err != nil {
		handler.logger.Error(fmt.Sprintf("PR reassign error: %s", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(entities.ResponseReassign{
		PullRequest: pr.ToResponse(),
		ReplacedBy:  userId,
	})
	if err != nil {
		handler.logger.Error(fmt.Sprintf("Failed to encode pr to json: %s", err))
		return
	}
}
