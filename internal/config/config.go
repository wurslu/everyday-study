package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	Environment     string
	DatabasePath    string
	VolcanoAPIKey   string
	VolcanoBaseURL  string
}

func Load() *Config {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Println("未找到 .env 文件，使用环境变量")
	}

	cfg := &Config{
		Port:           getEnv("PORT", "91"),
		Environment:    getEnv("ENVIRONMENT", "development"),
		DatabasePath:   getEnv("DATABASE_PATH", "learning.db"),
		VolcanoAPIKey:  getEnv("VOLCANO_API_KEY", ""),
		VolcanoBaseURL: getEnv("VOLCANO_BASE_URL", "https://ark.cn-beijing.volces.com/api/v3"),
	}

	if cfg.VolcanoAPIKey == "" {
		log.Fatal("VOLCANO_API_KEY 环境变量未设置")
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}