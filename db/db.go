package db

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model

	UUID              string
	Mail              string
	Phone             string
	TgId              uint
	Blocked           bool `gorm:"default: false"`
	LastBlockedReason string
	Quoata            uint

	CreatedAt time.Time
	UpdatedAt time.Time
}

func GetDB(dbp string) (*gorm.DB, error) {

	db, err := gorm.Open(sqlite.Open(dbp), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&User{})

	return db, nil
}
