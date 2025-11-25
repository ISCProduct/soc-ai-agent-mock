package config

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ConnectDB データベースに接続
func ConnectDB(cfg *Config) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
}
