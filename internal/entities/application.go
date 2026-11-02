package entities

import "time"

type ApplicationStatus string

const (
	StatusApplied ApplicationStatus = "applied"
	StatusFailed  ApplicationStatus = "failed"
)

type Application struct {
	ID        int64
	Vacancy   Vacancy
	Letter    string
	Status    ApplicationStatus
	Error     string
	AppliedAt time.Time
}
