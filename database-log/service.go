package database_log

import (
	"context"
	"time"
)

type Service interface {
	Create(logData, applicationID, userID string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo}
}

func (s *service) Create(logData, applicationID, userID string) error {
	currentTime := time.Now()
	data := StreamLog{
		Message:     logData,
		Source:      "DATABASE",
		Level:       "INFO",
		Timestamp:   currentTime.Format("2006-01-02 15:04:05"),
		CreatedAt:   currentTime,
		UpdatedAt:   currentTime,
		Process:     "DatabaseQueryProcess",
		Application: applicationID,
		User:        userID,
	}
	return s.repo.Create(context.Background(), data)
}
