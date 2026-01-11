package database

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func ConnectMySQL(dsn string) (*gorm.DB, error) {
	// Sesuai docs: gorm.Open(mysql.Open(dsn), &gorm.Config{})
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
