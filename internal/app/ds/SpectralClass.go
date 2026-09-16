package ds

import "time"

const (
	StatusPublished = "published"
	StatusDraft     = "draft"
	StatusDeleted   = "deleted"
)

// SpectralClass — «услуга» по теме: спектральный класс цефеид
type SpectralClass struct {
	ID          int       `gorm:"primaryKey"`
	Status      string    `gorm:"type:varchar(15);not null;default:'draft'"`
	Name        string    `gorm:"type:varchar(100);not null"`
	Description string    `gorm:"type:varchar(500)"`
	ImageKey    string    `gorm:"type:varchar(100)"`
	VideoKey    string    `gorm:"type:varchar(100)"`
	PlSlope     float64   `gorm:"not null;default:0"`
	PlIntercept float64   `gorm:"not null;default:0"`
	CreatorID   uint      `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Creator User `gorm:"foreignKey:CreatorID;constraint:OnDelete:RESTRICT"`
}

func (SpectralClass) TableName() string { return "spectral_classes" }
