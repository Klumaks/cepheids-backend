package ds

// Like — таблица м-м «пользователи ↔ спектральные классы»
type Like struct {
	ID              uint `gorm:"primaryKey"`
	UserID          uint `gorm:"not null;uniqueIndex:idx_user_class"`
	SpectralClassID int  `gorm:"not null;uniqueIndex:idx_user_class"`

	// Внешние ключи без каскадного удаления
	User          User          `gorm:"foreignKey:UserID;constraint:OnDelete:RESTRICT"`
	SpectralClass SpectralClass `gorm:"foreignKey:SpectralClassID;constraint:OnDelete:RESTRICT"`
}

func (Like) TableName() string { return "likes" }
