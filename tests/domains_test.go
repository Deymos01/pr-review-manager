package tests

import "time"

type CreatePRResponse struct {
	PR struct {
		PullRequestID     string   `json:"pull_request_id"`
		PullRequestName   string   `json:"pull_request_name"`
		AuthorID          string   `json:"author_id"`
		Status            string   `json:"status"`
		AssignedReviewers []string `json:"assigned_reviewers"`
	} `json:"pr"`
}

type GetReviewResponse struct {
	UserId       string `json:"user_id"`
	PullRequests []struct {
		PullRequestId   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorId        string `json:"author_id"`
		Status          string `json:"status"`
	} `json:"pull_requests"`
}

type ReassignUserResponse struct {
	Pr struct {
		PullRequestId     string   `json:"pull_request_id"`
		PullRequestName   string   `json:"pull_request_name"`
		AuthorId          string   `json:"author_id"`
		Status            string   `json:"status"`
		AssignedReviewers []string `json:"assigned_reviewers"`
	} `json:"pr"`
	ReplacedBy string `json:"replaced_by"`
}

type MergePRResponse struct {
	Pr struct {
		PullRequestId     string    `json:"pull_request_id"`
		PullRequestName   string    `json:"pull_request_name"`
		AuthorId          string    `json:"author_id"`
		Status            string    `json:"status"`
		AssignedReviewers []string  `json:"assigned_reviewers"`
		MergedAt          time.Time `json:"mergedAt"`
	} `json:"pr"`
}

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type Member struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type TeamResponse struct {
	Team struct {
		Name    string   `json:"team_name"`
		Members []Member `json:"members"`
	} `json:"team"`
}
