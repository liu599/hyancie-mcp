package logging

import (
	"log"
	"log/slog"
	"os"

	hyancie "github.com/liu599/hyancie"
)

var Logger *slog.Logger

// InitLogger initializes the global logger.
func InitLogger() error {
	logFilePath := hyancie.Config.Logging.FilePath
	if logFilePath == "" {
		logFilePath = "access.log" // Default value
	}

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := slog.NewJSONHandler(file, opts)
	Logger = slog.New(handler)

	// Redirect standard logger to the same file
	log.SetOutput(file)

	return nil
}
	log.SetOutput(file)

	return nil
}

