package config

import (
	"errors"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

// LoadDotenv loads variables from a .env file into the process environment.
// Missing .env files are non-fatal; OS-provided variables are used instead.
// Variables already set in the environment are not overwritten by godotenv.
func LoadDotenv(logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}

	if err := godotenv.Load(); err != nil {
		if isDotenvMissing(err) {
			logger.Info("no .env file loaded, using OS environment variables")
			return
		}
		logger.Warn("failed to load .env file, using OS environment variables", "error", err)
		return
	}

	logger.Info(".env file loaded")
}

func isDotenvMissing(err error) bool {
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	// godotenv returns a wrapped os.PathError when .env is absent.
	var pathErr *os.PathError
	return errors.As(err, &pathErr) && errors.Is(pathErr.Err, os.ErrNotExist)
}
