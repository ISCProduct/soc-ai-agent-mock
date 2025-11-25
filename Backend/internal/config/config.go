package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUser     string
	DBPass     string
	DBHost     string
	DBPort     string
	DBName     string
	ServerPort string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load() // .env があれば読み込む（本番では無視）

	cfg := &Config{
		DBUser: get("DB_USER", "app_user"),
		DBPass: get("DB_PASS", "app_pass"),
		// 明示的に IPv4 のループバックをデフォルトにする
		DBHost:     get("DB_HOST", "127.0.0.1"),
		DBPort:     get("DB_PORT", "3306"),
		DBName:     get("DB_NAME", "app_db"),
		ServerPort: get("SERVER_PORT", "80"),
	}

	return cfg, nil
}

// DSN は mysql ドライバ用の接続文字列を返す（例: user:pass@tcp(host:port)/dbname?parseTime=true&charset=utf8mb4&loc=Local）
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local",
		c.DBUser, c.DBPass, c.DBHost, c.DBPort, c.DBName)
}

func get(key, def string) string {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	return value
}
