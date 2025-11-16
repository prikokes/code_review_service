package entities

import "errors"

var (
	ErrPRAlreadyExists       = errors.New("PR already exists")
	ErrAuthorNotFound        = errors.New("author not found or inactive")
	ErrNotFound              = errors.New("PR not found")
	ErrPRAlreadyMerged       = errors.New("PR already merged")
	ErrUserIsNotAssignedToPR = errors.New("PR is not assigned to a user")
	ErrNoReplacement         = errors.New("no replacement found")
)

type ErrorStatsResponse struct {
	Error string `json:"error"`
}
