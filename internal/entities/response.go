package entities

import "time"

type ResponseReassign struct {
	PullRequest PullRequestDTO `json:"pr"`
	ReplacedBy  string         `json:"replaced_by"`
}

type ResponseMerge struct {
	PullRequest PullRequestDTO `json:"pr"`
	MergedAT    time.Time      `json:"merged_at"`
}

type ResponseCreatePR struct {
	PullRequest PullRequestDTO `json:"pr"`
}

type ResponseAddTeam struct {
	Team Team `json:"team"`
}

type ResponseSetIsActive struct {
	User User `json:"user"`
}

type PullRequestDTO struct {
	PullRequestID     string   `json:"pull_request_id"`
	PullRequestName   string   `json:"pull_request_name"`
	AuthorID          string   `json:"author_id"`
	Status            string   `json:"status"`
	AssignedReviewers []string `json:"assigned_reviewers,omitempty"`
}

func (pr PullRequest) ToResponse() PullRequestDTO {
	reviewerIDs := make([]string, len(pr.AssignedReviewers))
	for i, reviewer := range pr.AssignedReviewers {
		reviewerIDs[i] = reviewer.UserID
	}

	return PullRequestDTO{
		PullRequestID:     pr.PullRequestID,
		PullRequestName:   pr.PullRequestName,
		AuthorID:          pr.AuthorID,
		Status:            pr.Status,
		AssignedReviewers: reviewerIDs,
	}
}
