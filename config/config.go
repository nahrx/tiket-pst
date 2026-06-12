package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppPort        string
	AppEnv         string
	// PostgreSQL
	DBHost         string
	DBPort         string
	DBName         string
	DBUser         string
	DBPassword     string
	DBSSLMode      string
	DBDSN          string // computed DSN
	// JWT
	JWTSecret      string
	JWTExpiryHours int
	// Admin default
	AdminUsername  string
	AdminPassword  string
	// CORS
	AllowedOrigins []string
	// WA
	WANumber       string
}

var App *Config

func Load() {
	expiryHours, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))
	if err != nil {
		expiryHours = 24
	}

	originsRaw := getEnv("ALLOWED_ORIGINS", "http://localhost:8080")
	origins := []string{}
	for _, o := range strings.Split(originsRaw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}

	dbHost     := getEnv("DB_HOST",     "")
	dbPort     := getEnv("DB_PORT",     "")
	dbName     := getEnv("DB_NAME",     "")
	dbUser     := getEnv("DB_USER",     "")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbSSLMode  := getEnv("DB_SSLMODE",  "disable")

	// Gunakan DATABASE_URL jika ada (override semua field di atas)
	dsn := getEnv("DATABASE_URL", "")
	if dsn == "" {
		dsn = fmt.Sprintf(
			"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
			dbHost, dbPort, dbName, dbUser, dbPassword, dbSSLMode,
		)
	}

	App = &Config{
		AppPort:        getEnv("APP_PORT", "8080"),
		AppEnv:         getEnv("APP_ENV", "development"),
		DBHost:         dbHost,
		DBPort:         dbPort,
		DBName:         dbName,
		DBUser:         dbUser,
		DBPassword:     dbPassword,
		DBSSLMode:      dbSSLMode,
		DBDSN:          dsn,
		JWTSecret:      getEnv("JWT_SECRET", "default_secret_ganti_di_production"),
		JWTExpiryHours: expiryHours,
		AdminUsername:  getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:  getEnv("ADMIN_PASSWORD", "password"),
		AllowedOrigins: origins,
		WANumber:       getEnv("WA_NUMBER", ""),
	}

	if App.AppEnv == "production" && App.JWTSecret == "default_secret_ganti_di_production" {
		log.Fatal("[FATAL] JWT_SECRET harus diset di environment production!")
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
