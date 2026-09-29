package backend

import (
	"io"
	"log"
	"log/slog"
	"os"
	"sync/atomic"
)

type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelWarning
	LogLevelError
)

func (level LogLevel) String() string {
	switch level {
	case LogLevelInfo:
		return "Info"
	case LogLevelWarning:
		return "Warning"
	case LogLevelError:
		return "Error"
	default:
		return "Unknown"
	}
}

type EmbeddedLevelLogger struct {
	*log.Logger
	minimum atomic.Int32
}

func NewEmbeddedLevelLogger(output io.Writer, minimum LogLevel) *EmbeddedLevelLogger {
	logger := &EmbeddedLevelLogger{Logger: log.New(output, "", 0)}
	logger.SetLogLevel(minimum)
	return logger
}

func NewDefaultEmbeddedLevelLogger() *EmbeddedLevelLogger {
	return NewEmbeddedLevelLogger(os.Stderr, LogLevelError)
}

func (logger *EmbeddedLevelLogger) SetLogLevel(level LogLevel) {
	logger.minimum.Store(int32(level))
}

func (logger *EmbeddedLevelLogger) Log(level LogLevel, message string) {
	if level < LogLevel(logger.minimum.Load()) {
		return
	}
	logger.Logger.Println(level.String() + " " + message)
}

func (logger *EmbeddedLevelLogger) Infoln(message string) {
	logger.Log(LogLevelInfo, message)
}

func (logger *EmbeddedLevelLogger) Warnln(message string) {
	logger.Log(LogLevelWarning, message)
}

func (logger *EmbeddedLevelLogger) Errorln(message string) {
	logger.Log(LogLevelError, message)
}

func NewSlogLogger(output io.Writer, minimum slog.Leveler) *slog.Logger {
	return slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: minimum}))
}
