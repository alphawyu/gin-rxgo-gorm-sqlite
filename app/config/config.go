package api

import (
	"log/slog"
	"os"
	"strconv"
)

type Config struct {
	Database DatabaseConfig
}

type DatabaseConfig struct {
	Host         string
	Port         string
	MaxIdleConns int
	MaxOpenConns int
}

func NewConfig() (*Config, error) {
	var (
		err          error
		maxIdleConns = 2
		maxOpenConns = 150
	)
	poolSizeStr := os.Getenv("DATABASE_POOL_SIZE")
	if len(poolSizeStr) > 0 {
		maxIdleConns, err = strconv.Atoi(poolSizeStr)
		if err != nil {
			slog.Error("pool size is not valid, use default 2", "error", err)
		} else {
			maxOpenConns = maxIdleConns
		}
	}
	config := Config{
		Database: DatabaseConfig{
			Host: os.Getenv("DATABASE_HOST"),
			Port: os.Getenv("DATABASE_PORT"),
			MaxIdleConns: maxIdleConns,
			MaxOpenConns: maxOpenConns,
		},
	}

	return &config, nil
}

