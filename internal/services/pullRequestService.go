package services

import (
	"CodeRewievService/internal/entities"
	"errors"
	"gorm.io/gorm"
	"math/rand"
	"time"
)

type PullRequestService struct {
	db *gorm.DB
}

func NewPullRequestService(db *gorm.DB) *PullRequestService {
	return &PullRequestService{db: db}
}

func (prs *PullRequestService) Create(pr *entities.PullRequest) (*entities.PullRequest, error) {
	if pr == nil {
		return nil, errors.New("pull request cannot be nil")
	}

	if pr.PullRequestID == "" {
		return nil, errors.New("pull_request_id cannot be empty")
	}

	var existingPR entities.PullRequest
	result := prs.db.Where("pull_request_id = ?", pr.PullRequestID).First(&existingPR)
	if result.Error == nil {
		return nil, entities.ErrPRAlreadyExists
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	var author entities.User
	result = prs.db.Where("user_id = ? AND is_active = ?", pr.AuthorID, true).First(&author)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, entities.ErrAuthorNotFound
	} else if result.Error != nil {
		return nil, result.Error
	}

	var teamMembers []entities.User
	err := prs.db.Where("team_name = ? AND user_id != ? AND is_active = ?", author.TeamName, pr.AuthorID, true).Find(&teamMembers).Error
	if err != nil {
		return nil, err
	}

	// if len(teamMembers) == 0 {
	// return nil, errors.New("no active team members available for review")
	// }

	assignedReviewers := selectRandomReviewers(teamMembers, 2, pr.PullRequestID)
	println(len(assignedReviewers))

	newPR := &entities.PullRequest{
		PullRequestID:     pr.PullRequestID,
		PullRequestName:   pr.PullRequestName,
		AuthorID:          pr.AuthorID,
		Status:            "OPEN",
		AssignedReviewers: assignedReviewers,
	}

	err = prs.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(newPR).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	var createdPR entities.PullRequest
	err = prs.db.Preload("AssignedReviewers.User").
		Preload("Author").
		Where("pull_request_id = ?", pr.PullRequestID).
		First(&createdPR).Error
	if err != nil {
		return nil, err
	}

	return &createdPR, nil
}

func (prs *PullRequestService) Merge(prID string) (*entities.PullRequest, error) {
	if prID == "" {
		return nil, errors.New("pull_request_id cannot be empty")
	}

	var pr entities.PullRequest
	result := prs.db.Where("pull_request_id = ?", prID).First(&pr)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, entities.ErrNotFound
	} else if result.Error != nil {
		return nil, result.Error
	}

	if pr.Status == "MERGED" {
		return &pr, nil
	}

	now := time.Now()
	pr.Status = "MERGED"
	pr.MergedAt = &now

	if err := prs.db.Save(&pr).Error; err != nil {
		return nil, err
	}

	return &pr, nil
}

func selectRandomReviewers(users []entities.User, max int, pullRequestID string) []entities.PullRequestReviewer {
	if len(users) == 0 {
		return []entities.PullRequestReviewer{}
	}

	if len(users) <= max {
		max = len(users)
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	shuffled := make([]entities.User, len(users))
	copy(shuffled, users)
	r.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	reviewers := make([]entities.PullRequestReviewer, max)
	for i := 0; i < max; i++ {
		reviewers[i] = entities.PullRequestReviewer{
			PullRequestID: pullRequestID,
			UserID:        shuffled[i].UserID,
			AssignedAt:    time.Now(),
		}
	}

	return reviewers
}

func (prs *PullRequestService) Reassign(prID string, oldUserID string) (*entities.PullRequest, string, error) {
	if prID == "" {
		return nil, "", errors.New("pull_request_id cannot be empty")
	}
	if oldUserID == "" {
		return nil, "", errors.New("old_user_id cannot be empty")
	}

	var pr entities.PullRequest
	result := prs.db.Preload("AssignedReviewers").
		Where("pull_request_id = ?", prID).
		First(&pr)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, "", entities.ErrNotFound
	} else if result.Error != nil {
		return nil, "", result.Error
	}

	if pr.Status == "MERGED" {
		return nil, "", entities.ErrPRAlreadyMerged
	}

	found := false
	for _, reviewer := range pr.AssignedReviewers {
		if reviewer.UserID == oldUserID {
			found = true
			break
		}
	}
	if !found {
		return nil, "", entities.ErrUserIsNotAssignedToPR
	}

	var oldReviewer entities.User
	result = prs.db.Where("user_id = ?", oldUserID).First(&oldReviewer)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, "", entities.ErrNotFound
	} else if result.Error != nil {
		return nil, "", result.Error
	}

	var availableReviewers []entities.User
	query := prs.db.Where("team_name = ? AND user_id != ? AND is_active = ?",
		oldReviewer.TeamName, pr.AuthorID, true)

	for _, reviewer := range pr.AssignedReviewers {
		query = query.Where("user_id != ?", reviewer.UserID)
	}

	err := query.Find(&availableReviewers).Error
	if err != nil {
		return nil, "", err
	}

	if len(availableReviewers) == 0 {
		return nil, "", entities.ErrNoReplacement
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	newReviewer := availableReviewers[r.Intn(len(availableReviewers))]

	err = prs.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("pull_request_id = ? AND user_id = ?", prID, oldUserID).
			Delete(&entities.PullRequestReviewer{}).Error; err != nil {
			return err
		}

		newPRReviewer := entities.PullRequestReviewer{
			PullRequestID: prID,
			UserID:        newReviewer.UserID,
			AssignedAt:    time.Now(),
		}
		if err := tx.Create(&newPRReviewer).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, "", err
	}

	var updatedPR entities.PullRequest
	err = prs.db.Preload("AssignedReviewers.User").
		Preload("Author").
		Where("pull_request_id = ?", prID).
		First(&updatedPR).Error
	if err != nil {
		return nil, "", err
	}

	return &updatedPR, newReviewer.UserID, nil
}
