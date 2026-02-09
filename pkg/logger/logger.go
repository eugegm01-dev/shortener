package logger

import (
	"os"

	"github.com/rs/zerolog"
)

// Logger глобальный экземпляр логгера
var Logger zerolog.Logger

// Init инициализирует логгер
func Init() {
	// Устанавливаем уровень логирования на Info
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Создаем логгер с временными метками
	Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
}
