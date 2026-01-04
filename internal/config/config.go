package config

import (
	"os"
	"time"
	"strconv"
)

type Config struct {
	// GRPCPort is the port on which the gRPC server will listen.
	GRPCPort string

	// BackendAddress is the address of the backend service to which requests will be forwarded.
	BackendAddress string

	// CollapseWindow defines the time window for collapsing requests.
	CollapseWindow time.Duration

	// MaxBatchSize defines the maximum number of requests to collapse into a single backend request.
	MaxBatchSize int
}

func Load() *Config {
	return &Config{
		GRPCPort:       getEnv("GRPC_PORT", "50051"),
		BackendAddress: getEnv("BACKEND_ADDRESS", "localhost:8080"),
		CollapseWindow: getEnvAsDuration("COLLAPSE_WINDOW", 100*time.Millisecond),
		MaxBatchSize:   getEnvAsInt("MAX_BATCH_SIZE", 50),
	}
}

func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(name string, defaultVal int) int {
	if valueStr, exists := os.LookupEnv(name); exists {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultVal
}

func getEnvAsDuration(name string, defaultVal time.Duration) time.Duration {
	if valueStr, exists := os.LookupEnv(name); exists {
		if value, err := time.ParseDuration(valueStr); err == nil {
			return value
		}
	}
	return defaultVal
}	
