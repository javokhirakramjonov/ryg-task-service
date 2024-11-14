package model

import "time"

type Challenge struct {
	ID          int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	Title       string    `gorm:"type:varchar(255);not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	StartDate   time.Time `gorm:"type:timestamp;not null" json:"start_date"`
	EndDate     time.Time `gorm:"type:timestamp;not null" json:"end_date"`
	UserID      int32     `gorm:"not null;index;constraint:OnDelete:CASCADE;" json:"user_id"`
}

func (Challenge) TableName() string {
	return "challenges"
}
