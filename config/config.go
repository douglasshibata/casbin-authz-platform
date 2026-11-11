package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort     string
	DriverName     string
	DataSourceName string
	ModelPath      string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	LogLevel       string
}

func LoadConfig() Config {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	driver := os.Getenv("DB_DRIVER")
	if driver == "" {
		driver = "sqlite"
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		if driver == "postgres" {
			dsn = "host=localhost user=casbin password=casbin dbname=casbin port=5432 sslmode=disable"
		} else {
			dsn = "policies.db"
		}
	}

	modelPath := os.Getenv("CASBIN_MODEL_PATH")
	if modelPath == "" {
		modelPath = "config/model_rbac_tenants.conf"
	}

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = os.Getenv("REDIS_ADDR")
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	redisDB := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if parsed, err := strconv.Atoi(dbStr); err == nil {
			redisDB = parsed
		}
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	return Config{
		ServerPort:     port,
		DriverName:     driver,
		DataSourceName: dsn,
		ModelPath:      modelPath,
		RedisAddr:      redisAddr,
		RedisPassword:  redisPassword,
		RedisDB:        redisDB,
		LogLevel:       logLevel,
	}
}
