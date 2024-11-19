package model

import (
	"time"
)

const (
	TaskStatusNotStarted TaskStatus = "NOT_STARTED"
)

type TaskStatus string

type Task struct {
	ID          int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Title       string `gorm:"type:varchar(255);not null" json:"title"`
	Description string `gorm:"type:text" json:"description"`
	ChallengeID int64  `gorm:"not null;index;constraint:OnDelete:CASCADE;" json:"challenge_id"`
}

func (Task) TableName() string {
	return "tasks"
}

type TaskAndStatus struct {
	TaskID int64      `gorm:"primaryKey;not null" json:"task_id"`
	Date   time.Time  `gorm:"primaryKey;type:date;not null" json:"date"`
	Status TaskStatus `gorm:"type:varchar(20);not null;check:status IN ('NOT_STARTED', 'COMPLETED', 'NOT_COMPLETED')" json:"status"`

	Task Task `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE;" json:"task"`
}

func (TaskAndStatus) TableName() string {
	return "task_and_status"
}
