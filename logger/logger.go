package logger

import (
	"os"

	"github.com/rs/zerolog"
)

var Logger zerolog.Logger

func InitLog() {
	runLogFile, _ := os.OpenFile(
		"aws-commander.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0664,
	)
	consoleWriter := zerolog.ConsoleWriter{Out: runLogFile}
	Logger = zerolog.New(consoleWriter).With().Timestamp().Logger()
}
