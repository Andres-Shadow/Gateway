package database

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func Open(sqlitePath string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{})
}

func AutoMigrate(db *gorm.DB, models ...any) error {
	return db.AutoMigrate(models...)
}
