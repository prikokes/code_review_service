package services

import (
	"CodeRewievService/internal/entities"
	"errors"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{
		db: db,
	}
}

func (us *UserService) SetIsActive(user *entities.User) (*entities.User, error) {
	if user == nil {
		return nil, errors.New("user cannot be nil")
	}

	if user.UserID == "" {
		return nil, errors.New("user_id cannot be empty")
	}

	var existingUser entities.User
	result := us.db.Where("user_id = ?", user.UserID).First(&existingUser)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("user not found")
	} else if result.Error != nil {
		return nil, result.Error
	}

	existingUser.IsActive = user.IsActive

	if err := us.db.Save(&existingUser).Error; err != nil {
		return nil, err
	}

	return &existingUser, nil
}

func (us *UserService) GetReview(userID string) (*entities.UserReview, error) {
	if userID == "" {
		return nil, errors.New("user_id cannot be empty")
	}

	var user entities.User
	result := us.db.Where("user_id = ?", userID).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("user not found")
	} else if result.Error != nil {
		return nil, result.Error
	}

	var pullRequests []entities.PullRequest
	err := us.db.
		Joins("JOIN pull_request_reviewers ON pull_requests.pull_request_id = pull_request_reviewers.pull_request_id").
		Where("pull_request_reviewers.user_id = ?", userID).
		Find(&pullRequests).Error
	if err != nil {
		return nil, err
	}

	pullRequestsShort := make([]entities.PullRequestDTO, len(pullRequests))
	for i, pr := range pullRequests {
		pullRequestsShort[i] = pr.ToResponse()
	}

	userReview := &entities.UserReview{
		UserID:       userID,
		PullRequests: pullRequestsShort,
	}

	return userReview, nil
}
