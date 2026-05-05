// Package logger provides a global zerolog logger with timestamp and info level.
package logger

import (
	"os"

	"github.com/rs/zerolog"
)

// Logger is the global logger instance used throughout the application.
var Logger zerolog.Logger

// Init initialises the global logger with human‑readable timestamps
// and sets the global logging level to Info.
func Init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
}
