package config

import "os"

type Config struct {
	ServerPort    string
	DatabasePath  string
	FrontendDir   string
	EnableCors    bool
	PlaywrightPath string
}

func Load() *Config {
	return &Config{
		ServerPort:    getEnv("SERVER_PORT", "8888"),
		DatabasePath:  getEnv("DATABASE_PATH", "./db/getjobs.db"),
		FrontendDir:   getEnv("FRONTEND_DIR", "./front"),
		EnableCors:    getEnv("ENABLE_CORS", "true") == "true",
		PlaywrightPath: getEnv("PLAYWRIGHT_PATH", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
