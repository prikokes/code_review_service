package entities

type UserStats struct {
	UserID            string `json:"user_id"`
	Username          string `json:"username"`
	AuthoredPRs       int64  `json:"authored_prs"`
	OpenAuthoredPRs   int64  `json:"open_authored_prs"`
	MergedAuthoredPRs int64  `json:"merged_authored_prs"`
	AssignedReviews   int64  `json:"assigned_reviews"`
}

type TeamStats struct {
	TeamName          string  `json:"team_name"`
	TotalMembers      int64   `json:"total_members"`
	ActiveMembers     int64   `json:"active_members"`
	TotalPRs          int64   `json:"total_prs"`
	OpenPRs           int64   `json:"open_prs"`
	MergedPRs         int64   `json:"merged_prs"`
	AvgMergeTimeHours float64 `json:"avg_merge_time_hours"`
}

type MergeTimeStats struct {
	Period               string  `json:"period"`
	Date                 string  `json:"date"`
	TotalMerged          int64   `json:"total_merged"`
	AvgMergeTimeHours    float64 `json:"avg_merge_time_hours"`
	MedianMergeTimeHours float64 `json:"median_merge_time_hours"`
}
