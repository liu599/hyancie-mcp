package logging

import (
	"log"
	"log/slog"
	"os"

	_ "github.com/liu599/hyancie"
)

var (
	Logger *slog.Logger
)

// InitLogger initializes the global logger.
func InitLogger() error {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	Logger = slog.New(handler)

	// Redirect standard logger to the same file
	log.SetOutput(os.Stdout)

	return nil
}

