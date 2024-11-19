package model

import "time"

const (
	ChallengeStatusDraft    = "DRAFT"
	ChallengeStatusStarted  = "STARTED"
	ChallengeStatusFinished = "FINISHED"
	MaxChallengeDays        = 7
)

type Challenge struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Title       string    `gorm:"type:varchar(255);not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	StartDate   time.Time `gorm:"type:timestamp;" json:"start_date"`
	EndDate     time.Time `gorm:"type:timestamp;" json:"end_date"`
	UserID      int64     `gorm:"not null;index;constraint:OnDelete:CASCADE;" json:"user_id"`
	Status      string    `gorm:"type:varchar(20);not null;check:status IN ('DRAFT', 'STARTED', 'FINISHED')" json:"status"`
	Days        int32     `gorm:"type:int" json:"days"`
}

func (Challenge) TableName() string {
	return "challenges"
}
