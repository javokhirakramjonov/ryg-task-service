package model

import "time"

const (
	ChallengeStatusDraft    = "DRAFT"
	ChallengeStatusStarted  = "STARTED"
	ChallengeStatusFinished = "FINISHED"

	ChallengeInvitationStatusPending  = "PENDING"
	ChallengeInvitationStatusAccepted = "ACCEPTED"

	ChallengeAndUserOwnerRole       = "OWNER"
	ChallengeAndUserParticipantRole = "PARTICIPANT"

	MaxChallengeDays = 7
)

type Challenge struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Title       string    `gorm:"type:varchar(255);not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	StartDate   time.Time `gorm:"type:timestamp;" json:"start_date"`
	EndDate     time.Time `gorm:"type:timestamp;" json:"end_date"`
	Status      string    `gorm:"type:varchar(20);not null;check:status IN ('DRAFT', 'STARTED', 'FINISHED')" json:"status"`
	Days        int32     `gorm:"type:int" json:"days"`
}

func (Challenge) TableName() string {
	return "challenges"
}

type ChallengeAndUser struct {
	ChallengeID int64  `gorm:"primaryKey" json:"challenge_id"`
	UserID      int64  `gorm:"primaryKey" json:"user_id"`
	UserRole    string `gorm:"type:varchar(20);not null;check:user_role IN ('OWNER', 'PARTICIPANT')" json:"user_role"`

	Challenge Challenge `gorm:"foreignKey:ChallengeID;references:ID;constraint:OnDelete:CASCADE" json:"challenge"`
}

type ChallengeInvitation struct {
	ChallengeID int64  `gorm:"primaryKey" json:"challenge_id"`
	UserID      int64  `gorm:"primaryKey" json:"user_id"`
	Status      string `gorm:"type:varchar(20);not null;check:status IN ('PENDING', 'ACCEPTED')" json:"status"`

	Challenge Challenge `gorm:"foreignKey:ChallengeID;references:ID;constraint:OnDelete:CASCADE" json:"challenge"`
}
