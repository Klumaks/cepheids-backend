package ds

import "time"

const (
	StatusPublished = "published"
	StatusDraft     = "draft"
	StatusDeleted   = "deleted"
)

type SpectralClass struct {
	ID          int        `gorm:"primaryKey" json:"id"`
	Status      string     `gorm:"type:varchar(15);not null;default:'draft'" json:"-"`
	Name        string     `gorm:"type:varchar(100);not null" json:"name"`
	Description string     `gorm:"type:varchar(500)" json:"description"`
	ImageKey    string     `gorm:"type:varchar(100)" json:"image_key"`
	VideoKey    string     `gorm:"type:varchar(100)" json:"video_key"`
	PlSlope     *float64   `gorm:"default:null" json:"pl_slope"`
	PlIntercept *float64   `gorm:"default:null" json:"pl_intercept"`
	CreatorID   uint       `gorm:"not null" json:"creator_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	Creator User `gorm:"foreignKey:CreatorID;constraint:OnDelete:RESTRICT" json:"-"`

	// Дополнительные поля для API ответов
	ImageURL   string `gorm:"-" json:"image_url"`
	VideoURL   string `gorm:"-" json:"video_url"`
	LikesCount int64  `gorm:"-" json:"likes_count"`
	IsMine     bool   `gorm:"-" json:"is_mine"`
	IsLiked    bool   `gorm:"-" json:"is_liked"`
}

func (SpectralClass) TableName() string { return "spectral_classes" }
