package interfaces

import "CodeRewievService/internal/entities"

type UserServiceInterface interface {
	SetIsActive(user *entities.User) (*entities.User, error)
	GetReview(userID string) (*entities.UserReview, error)
}

type TeamServiceInterface interface {
	Add(team *entities.Team) error
	Get(teamName string) (*entities.Team, error)
	MassDeactivateTeamUsers(teamName string) error
}

type PullRequestServiceInterface interface {
	Create(PullRequest *entities.PullRequest) (*entities.PullRequest, error)
	Reassign(prID string, userID string) (*entities.PullRequest, string, error)
	Merge(prID string) (*entities.PullRequest, error)
}

type StatsServiceInterface interface {
	GetTeamStats(teamName string) (*entities.TeamStats, error)
	GetUserStats(teamName string) ([]entities.UserStats, error)
	TeamExists(teamName string) (bool, error)
}
