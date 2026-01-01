package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerAddr string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

func Load() *Config {
	_ = godotenv.Load()

	// Support PORT env var (used by Render, Heroku, etc.)
	// If PORT is set, use it; otherwise check SERVER_ADDR; otherwise default to :8080
	port := getEnv("PORT", "")
	serverAddr := getEnv("SERVER_ADDR", "")
	if port != "" {
		serverAddr = ":" + port
	} else if serverAddr == "" {
		serverAddr = ":8080"
	}

	cfg := &Config{
		ServerAddr: serverAddr,
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "starpi"),
	}

	log.Println("Config loaded")
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
