package testcfg

import (
	"log/slog"
	"os"
	"testing"
)

// Test configuration (environment variables)
const (
	// log level, e.g. ERROR, WARN, INFO, DEBUG
	logLevel = "LOG_LEVEL"
)

func init() {
	var level slog.Level
	if err := level.UnmarshalText([]byte(os.Getenv(logLevel))); err == nil {
		slog.SetLogLoggerLevel(level)
	}
}

// Config reads a test configuration. Test is skipped, if the environment
// variable with the specified name is not found.
func Config(t *testing.T, name string) string {
	v, ok := os.LookupEnv(name)
	if !ok {
		t.Skip("Environment variable " + name + " not set")
	}
	return v
}
