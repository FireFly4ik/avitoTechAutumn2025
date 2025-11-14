package gorm

import (
	"time"
)

// PullRequest - модель БД для pull request
type PullRequest struct {
	PullRequestID   string     `gorm:"column:pull_request_id;primaryKey"`
	PullRequestName string     `gorm:"column:pull_request_name;not null"`
	AuthorID        string     `gorm:"column:author_id;not null"`
	Status          string     `gorm:"column:status;not null;default:OPEN"`
	CreatedAt       time.Time  `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	MergedAt        *time.Time `gorm:"column:merged_at"`
}

func (PullRequest) TableName() string {
	return "pull_requests"
}

// User - модель БД для пользователя
type User struct {
	UserID   string `gorm:"column:user_id;primaryKey"`
	Username string `gorm:"column:username;not null"`
	TeamName string `gorm:"column:team_name;not null"`
	IsActive bool   `gorm:"column:is_active;not null;default:true"`
}

func (User) TableName() string {
	return "users"
}

// Team - модель БД для команды
type Team struct {
	TeamName string `gorm:"column:team_name;primaryKey"`
	Members  []User `gorm:"foreignKey:TeamName;references:TeamName"`
}

func (Team) TableName() string {
	return "teams"
}

// Reviewer - модель БД для связи PR и ревьюверов
type Reviewer struct {
	PullRequestID string `gorm:"column:pull_request_id;primaryKey"`
	ReviewerID    string `gorm:"column:reviewer_id;primaryKey"`
}

func (Reviewer) TableName() string {
	return "pull_request_reviewers"
}
