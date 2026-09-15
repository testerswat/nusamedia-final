package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                   int
	DatabaseURL, JWTSecret string
}

func Load() Config {
	p, _ := strconv.Atoi(os.Getenv("PORT"))
	if p == 0 {
		p = 8080
	}
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "change-me-in-production"
	}
	return Config{Port: p, DatabaseURL: os.Getenv("DATABASE_URL"), JWTSecret: s}
}
