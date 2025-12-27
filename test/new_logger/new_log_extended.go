package new_logger

import (
	"log"
	"os"
)

type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelWarning
	LogLevelError
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelInfo:
		return "Info"
	case LogLevelWarning:
		return "Warning"
	case LogLevelError:
		return "Error"
	}
	return "Unknown"
}

type LogExtended struct {
	*log.Logger
	logLevel LogLevel
}

func NewLogExtended() *LogExtended {
	return &LogExtended{
		Logger:   log.New(os.Stderr, "", log.LstdFlags),
		logLevel: LogLevelError,
	}
}

func (l *LogExtended) println(srcLogLvl LogLevel, prefix, msg string) {
	if l.logLevel > srcLogLvl {
		return
	}

	l.Logger.Println(prefix + msg)
}

func (l *LogExtended) SetLogLevel(level LogLevel) {
	l.logLevel = level
}

func (l *LogExtended) Infoln(msg string) {
	l.println(LogLevelInfo, LogLevelInfo.String()+" ", msg)
}

func (l *LogExtended) Warnln(msg string) {
	l.println(LogLevelWarning, LogLevelWarning.String()+" ", msg)
}

func (l *LogExtended) Errorln(msg string) {
	l.println(LogLevelError, LogLevelError.String()+" ", msg)
}
