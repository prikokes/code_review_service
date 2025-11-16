package services

import (
	"CodeRewievService/internal/entities"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type TeamService struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewTeamService(db *gorm.DB, logger *slog.Logger) *TeamService {
	return &TeamService{
		db:     db,
		logger: logger,
	}
}

func (ts *TeamService) Add(team *entities.Team) error {
	if team == nil {
		return errors.New("team cannot be nil")
	}

	if team.TeamName == "" {
		return errors.New("team name cannot be empty")
	}

	var existingTeam entities.Team
	result := ts.db.Where("team_name = ?", team.TeamName).First(&existingTeam)
	if result.Error == nil {
		return errors.New("team already exists")
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}

	return ts.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(team).Error; err != nil {
			return err
		}

		for _, member := range team.Members {
			user := &entities.User{
				UserID:   member.UserID,
				Username: member.Username,
				TeamName: team.TeamName,
				IsActive: member.IsActive,
			}

			if err := tx.Save(user).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (ts *TeamService) Get(teamName string) (*entities.Team, error) {
	if teamName == "" {
		return nil, errors.New("team name cannot be empty")
	}

	fmt.Printf("Searching for team: '%s'\n", teamName)

	var team entities.Team

	result := ts.db.Debug().Where("team_name = ?", teamName).First(&team)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		fmt.Printf("Team '%s' not found in database\n", teamName)

		var allTeams []entities.Team
		ts.db.Debug().Find(&allTeams)
		fmt.Printf("Available teams: %+v\n", allTeams)

		return nil, errors.New("team not found")
	} else if result.Error != nil {
		fmt.Printf("Database error: %v\n", result.Error)
		return nil, result.Error
	}

	fmt.Printf("Found team: %+v\n", team)

	var users []entities.User
	if err := ts.db.Debug().Where("team_name = ?", team.TeamName).Find(&users).Error; err != nil {
		fmt.Printf("Error loading users: %v\n", err)
		return nil, err
	}

	fmt.Printf("Found %d users for team\n", len(users))

	team.Members = users

	return &team, nil
}

func (ts *TeamService) MassDeactivateTeamUsers(teamName string) error {
	startTime := time.Now()

	return ts.db.Transaction(func(tx *gorm.DB) error {
		var team entities.Team
		if err := tx.Where("team_name = ?", teamName).First(&team).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("team not found")
			}
			return err
		}

		result := tx.Model(&entities.User{}).
			Where("team_name = ? AND is_active = ?", teamName, true).
			Update("is_active", false)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return nil
		}

		if err := ts.removeReviewersFromTeamOpenPRs(tx, teamName); err != nil {
			return err
		}

		if time.Since(startTime) > 100*time.Millisecond {
			ts.logger.Warn("MassDeactivateTeamUsers execution time exceeded 100 ms", "team", teamName, "duration", time.Since(startTime))
		}

		return nil
	})
}

func (ts *TeamService) removeReviewersFromTeamOpenPRs(tx *gorm.DB, teamName string) error {
	var openPRs []entities.PullRequest
	if err := tx.Joins("JOIN users ON pull_requests.author_id = users.user_id").
		Where("users.team_name = ? AND pull_requests.status = ?", teamName, "OPEN").
		Find(&openPRs).Error; err != nil {
		return err
	}

	if len(openPRs) == 0 {
		return nil
	}

	prIDs := make([]string, len(openPRs))
	for i, pr := range openPRs {
		prIDs[i] = pr.PullRequestID
	}

	if err := tx.Where("pull_request_id IN (?)", prIDs).
		Delete(&entities.PullRequestReviewer{}).Error; err != nil {
		return err
	}

	return nil
}
