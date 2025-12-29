package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Port          string
	DBDSN         string
	JWTSecret     string
	AllowedOrigin string
	RateLimitRPS  float64
	RateLimitBurst int
}

func Load() Config {
	cfg := Config{
		Port:          getEnv("PORT", "8080"),
		DBDSN:         getEnv("DB_DSN", "root:Jim2002@tcp(127.0.0.1:3306)/P2P_Chat?parseTime=true"),
		JWTSecret:     getEnv("JWT_SECRET", "CacHeThongPhanTanMaster2025"),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "http://localhost:5173"),
		RateLimitRPS:  getEnvFloat("RATE_LIMIT_RPS", 5),
		RateLimitBurst: getEnvInt("RATE_LIMIT_BURST", 10),
	}
	if cfg.JWTSecret == "dev_secret_change_me" {
		log.Println("warning: using default JWT secret, change in production")
	}
	return cfg
}

func getEnv(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

func getEnvInt(key string, def int) int {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return i
}

func getEnvFloat(key string, def float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return def
	}
	return f
}
