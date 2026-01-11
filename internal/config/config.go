package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                 int
	DBDSN                string
	JWTSecret            string
	WSInsecureSkipVerify bool
}

func Load() Config {
	port := 8084
	if v := os.Getenv("APP_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}

	wsInsecure := false
	if os.Getenv("WS_INSECURE_SKIP_VERIFY") == "true" {
		wsInsecure = true
	}

	return Config{
		Port:                 port,
		DBDSN:                os.Getenv("DB_DSN"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		WSInsecureSkipVerify: wsInsecure,
	}
}