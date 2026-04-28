package logger

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type Logger = zerolog.Logger

func New(level string) Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	parsedLevel, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		parsedLevel = zerolog.InfoLevel
	}

	return logger.Level(parsedLevel)
}
