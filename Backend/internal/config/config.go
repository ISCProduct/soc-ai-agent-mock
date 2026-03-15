package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUser          string
	DBPass          string
	DBHost          string
	DBPort          string
	DBName          string
	ServerPort      string
	GBizInfoBaseURL string
	GBizInfoToken   string
}

func LoadConfig() (*Config, error) {
	env := os.Getenv("APP_ENV")

	if env != "production" {
		// ローカル開発環境では .env ファイルを読み込む
		err := godotenv.Load()
		if err != nil {
			log.Println("Warning: .env file not found. Skipping.")
		}
	}

	cfg := &Config{
		DBUser:          os.Getenv("DB_USER"),
		DBPass:          os.Getenv("DB_PASSWORD"),
		DBHost:          os.Getenv("DB_HOST"),
		DBPort:          os.Getenv("DB_PORT"),
		DBName:          os.Getenv("DB_NAME"),
		ServerPort:      get("SERVER_PORT", "80"),
		GBizInfoBaseURL: get("GBIZINFO_BASE_URL", ""),
		GBizInfoToken:   get("GBIZINFO_API_TOKEN", ""),
	}

	// 必須値チェック
	if cfg.DBHost == "" || cfg.DBPort == "" || cfg.DBUser == "" || cfg.DBPass == "" || cfg.DBName == "" {
		log.Fatal("Missing one or more required environment variables for database connection")
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
