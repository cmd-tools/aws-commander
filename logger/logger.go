package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

var Logger zerolog.Logger

const awsCommanderLogFileName = "aws-commander.log"

// LogFilePath returns the full path to the log file in the system temporary directory.
func LogFilePath() string {
	return filepath.Join(os.TempDir(), awsCommanderLogFileName)
}

var LogChannel chan string

func InitLog(isLogViewEnabled bool) {
	runLogFile, _ := os.OpenFile(
		LogFilePath(),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0664,
	)

	consoleWriter := io.MultiWriter(runLogFile)

	if isLogViewEnabled {
		// use buffered channel with max 65534, ideally we won't log that amount of lines in a really short time
		LogChannel = make(chan string, ^uint16(0))
		consoleWriter = io.MultiWriter(runLogFile, &channelWriter{LogChannel})
	}

	Logger = zerolog.New(consoleWriter).With().Timestamp().Logger()
}

type channelWriter struct {
	channel chan string
}

func (w *channelWriter) Write(p []byte) (n int, err error) {
	w.channel <- string(p)
	return len(p), nil
}
