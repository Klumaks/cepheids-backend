package ds

import "time"

const (
	StatusPublished = "published"
	StatusDraft     = "draft"
	StatusDeleted   = "deleted"
)

// SpectralClass — «услуга» по теме: спектральный класс цефеид
type SpectralClass struct {
	ID          int       `gorm:"primaryKey"`                                    // NOT NULL, PK
	Status      string    `gorm:"type:varchar(15);not null;default:'draft'"`     // NOT NULL
	Name        string    `gorm:"type:varchar(100);not null"`                    // NOT NULL
	Description string    `gorm:"type:varchar(500)"`                             // NULL (необязательное)
	ImageKey    string    `gorm:"type:varchar(100)"`                             // NULL (url, необязательное)
	VideoKey    string    `gorm:"type:varchar(100)"`                             // NULL (url, необязательное)
	PlSlope     *float64                                                        // NULL (поле по теме, необязательное)
	PlIntercept *float64                                                        // NULL (поле по теме, необязательное)
	CreatorID   uint      `gorm:"not null"`                                     // NOT NULL, FK на users
	CreatedAt   time.Time                                                       // NOT NULL (дата создания, авто)
	UpdatedAt   time.Time                                                       // NULL (дата формирования, авто)

	Creator User `gorm:"foreignKey:CreatorID;constraint:OnDelete:RESTRICT"`
}

func (SpectralClass) TableName() string { return "spectral_classes" }
