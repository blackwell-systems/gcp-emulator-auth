package emulatorauth

import "os"

// Config holds IAM emulator configuration
type Config struct {
	// Mode is the authorization mode (off, permissive, strict)
	Mode AuthMode

	// Host is the IAM emulator gRPC endpoint (host:port)
	Host string

	// Trace enables IAM decision logging
	Trace bool
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() Config {
	return Config{
		Mode:  ParseAuthMode(os.Getenv("IAM_MODE")),
		Host:  getEnvWithDefault("IAM_EMULATOR_HOST", "localhost:8080"),
		Trace: os.Getenv("IAM_TRACE") == "true",
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
