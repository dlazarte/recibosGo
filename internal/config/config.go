package config

import (
	"os"
)

type Config struct {
	Port          string
	DatabaseURL   string
	SessionSecret string
	AdminEmail    string
	AdminPassword string
	CSVPath       string
}

func Load() Config {
	return Config{
		Port:          getEnv("PORT", "8090"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://recibosgo:recibosgo@localhost:5432/recibosgo?sslmode=disable"),
		SessionSecret: getEnv("SESSION_SECRET", "dev-secret-cambiar-en-produccion"),
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@recibosgo.local"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		CSVPath:       getEnv("CSV_PATH", "dptos_dueños_inquilinos.csv"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
