package services

import (
	"CodeRewievService/internal/entities"
	"gorm.io/gorm"
)

type StatsService struct {
	db *gorm.DB
}

func NewStatsService(db *gorm.DB) *StatsService {
	return &StatsService{
		db: db,
	}
}

func (s *StatsService) GetTeamStats(teamName string) (*entities.TeamStats, error) {
	var stats entities.TeamStats
	stats.TeamName = teamName

	if err := s.db.Model(&entities.User{}).Where("team_name = ?", teamName).Count(&stats.TotalMembers).Error; err != nil {
		return nil, err
	}

	if err := s.db.Model(&entities.User{}).Where("team_name = ? AND is_active = true", teamName).Count(&stats.ActiveMembers).Error; err != nil {
		return nil, err
	}

	if err := s.db.Model(&entities.PullRequest{}).
		Joins("JOIN users ON pull_requests.author_id = users.user_id").
		Where("users.team_name = ?", teamName).
		Count(&stats.TotalPRs).Error; err != nil {
		return nil, err
	}

	if err := s.db.Model(&entities.PullRequest{}).
		Joins("JOIN users ON pull_requests.author_id = users.user_id").
		Where("users.team_name = ? AND pull_requests.status = 'OPEN'", teamName).
		Count(&stats.OpenPRs).Error; err != nil {
		return nil, err
	}

	if err := s.db.Model(&entities.PullRequest{}).
		Joins("JOIN users ON pull_requests.author_id = users.user_id").
		Where("users.team_name = ? AND pull_requests.status = 'MERGED'", teamName).
		Count(&stats.MergedPRs).Error; err != nil {
		return nil, err
	}

	var avgMergeTime struct {
		AvgHours float64
	}

	err := s.db.Table("pull_requests").
		Joins("JOIN users ON pull_requests.author_id = users.user_id").
		Where("users.team_name = ? AND pull_requests.status = 'MERGED' AND pull_requests.merged_at IS NOT NULL", teamName).
		Select("AVG(EXTRACT(EPOCH FROM (merged_at - created_at))/3600) as avg_hours").
		Scan(&avgMergeTime).Error

	if err != nil {

	} else {
		stats.AvgMergeTimeHours = avgMergeTime.AvgHours
	}

	return &stats, nil
}

func (s *StatsService) GetUserStats(teamName string) ([]entities.UserStats, error) {
	var users []entities.User
	if err := s.db.Where("team_name = ?", teamName).Find(&users).Error; err != nil {
		return nil, err
	}

	var userStats []entities.UserStats

	for _, user := range users {
		var stats entities.UserStats
		stats.UserID = user.UserID
		stats.Username = user.Username

		// PR созданные пользователем
		s.db.Model(&entities.PullRequest{}).Where("author_id = ?", user.UserID).Count(&stats.AuthoredPRs)
		s.db.Model(&entities.PullRequest{}).Where("author_id = ? AND status = 'OPEN'", user.UserID).Count(&stats.OpenAuthoredPRs)
		s.db.Model(&entities.PullRequest{}).Where("author_id = ? AND status = 'MERGED'", user.UserID).Count(&stats.MergedAuthoredPRs)

		// PR где пользователь ревьюер
		s.db.Model(&entities.PullRequestReviewer{}).Where("user_id = ?", user.UserID).Count(&stats.AssignedReviews)

		userStats = append(userStats, stats)
	}

	return userStats, nil
}

func (s *StatsService) TeamExists(teamName string) (bool, error) {
	var exists bool
	err := s.db.Model(&entities.Team{}).
		Select("count(*) > 0").
		Where("team_name = ?", teamName).
		Find(&exists).Error
	return exists, err
}
