package ds

type Like struct {
	ID              uint `gorm:"primaryKey" json:"id"`
	UserID          uint `gorm:"not null;uniqueIndex:idx_user_class" json:"user_id"`
	SpectralClassID int  `gorm:"not null;uniqueIndex:idx_user_class" json:"spectral_class_id"`

	User          User          `gorm:"foreignKey:UserID;constraint:OnDelete:RESTRICT" json:"-"`
	SpectralClass SpectralClass `gorm:"foreignKey:SpectralClassID;constraint:OnDelete:RESTRICT" json:"-"`
}

func (Like) TableName() string { return "likes" }
