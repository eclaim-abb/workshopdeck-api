package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	JWTSecret    string
	Port         string
	FrontendURLs []string
	Env          string
	GinMode      string
	UploadPath   string
	BaseURL      string
}

func (c *Config) Validate() error {
	required := map[string]string{
		"JWT_SECRET":  c.JWTSecret,
		"DB_HOST":     c.DBHost,
		"DB_USER":     c.DBUser,
		"DB_PASSWORD": c.DBPassword,
		"DB_NAME":     c.DBName,
	}

	for key, val := range required {
		if val == "" {
			return fmt.Errorf("required environment variable %s is not set", key)
		}
	}

	if c.JWTSecret == "super-secret-jwt-lmao" && c.Env == "production" {
		return fmt.Errorf("JWT_SECRET appears to be the default value — do not use in production")
	}

	return nil
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	frontendURLsStr := getEnv("FRONTEND_URLS", "http://localhost:3000")
	frontendURLs := strings.Split(frontendURLsStr, ",")

	for i, url := range frontendURLs {
		frontendURLs[i] = strings.TrimSpace(url)
	}

	return &Config{
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "3306"),
		DBUser:       getEnv("DB_USER", "root"),
		DBPassword:   getEnv("DB_PASSWORD", ""),
		DBName:       getEnv("DB_NAME", "workshop_deck_2025"),
		JWTSecret:    getEnv("JWT_SECRET", "super-secret-jwt-lmao"),
		Port:         getEnv("PORT", "8080"),
		FrontendURLs: frontendURLs,
		Env:          getEnv("APP_ENV", "development"),
		GinMode:      getEnv("GIN_MODE", "debug"),
		UploadPath:   getEnv("UPLOAD_PATH", "./uploads"),
		BaseURL:      getEnv("BASE_URL", "http://localhost:8080"),
	}
}

func LoadSecondaryConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	frontendURLsStr := getEnv("FRONTEND_URLS", "http://localhost:3000")
	frontendURLs := strings.Split(frontendURLsStr, ",")

	for i, url := range frontendURLs {
		frontendURLs[i] = strings.TrimSpace(url)
	}

	return &Config{
		DBHost:       getEnv("DB_ALT_HOST", "localhost"),
		DBPort:       getEnv("DB_ALT_PORT", "3306"),
		DBUser:       getEnv("DB_ALT_USER", "root"),
		DBPassword:   getEnv("DB_ALT_PASSWORD", ""),
		DBName:       getEnv("DB_ALT_NAME", "workshop_deck_2025"),
		JWTSecret:    getEnv("JWT_SECRET", "super-secret-jwt-lmao"),
		Port:         getEnv("PORT", "8080"),
		FrontendURLs: frontendURLs,
		Env:          getEnv("APP_ENV", "development"),
		GinMode:      getEnv("GIN_MODE", "debug"),
		UploadPath:   getEnv("UPLOAD_PATH", "./uploads"),
		BaseURL:      getEnv("BASE_URL", "http://localhost:8080"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
