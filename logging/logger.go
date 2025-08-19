package logging

import (
	"log"
	"log/slog"
	"os"

	hyancie "github.com/liu599/hyancie"
)

var (
	Logger  *slog.Logger
	logFile *os.File
)

// InitLogger initializes the global logger.
func InitLogger() error {
	logFilePath := hyancie.Config.Logging.FilePath
	if logFilePath == "" {
		logFilePath = "access.log" // Default value
	}

	var err error
	logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := slog.NewJSONHandler(logFile, opts)
	Logger = slog.New(handler)

	// Redirect standard logger to the same file
	log.SetOutput(logFile)

	return nil
}

// Close closes the log file.
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}
