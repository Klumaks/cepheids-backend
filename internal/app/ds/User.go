package ds

type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Login    string `gorm:"type:varchar(25);unique;not null" json:"login"`
	Password string `gorm:"type:varchar(100);not null" json:"-"`
}

func (User) TableName() string { return "users" }
