package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// LogLevel type for defining different log levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// Logger interface with methods for various log levels
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
}

type LoggerInstance struct {
	loggerImpl
}

// loggerImpl implements the Logger interface
type loggerImpl struct {
	logLevel LogLevel
}

// NewLogger returns a new instance of Logger with the specified log level
func NewLogger() *LoggerInstance {

	logLevel := getLogLevelForEnvironment()
	return &LoggerInstance{
		loggerImpl: loggerImpl{
			logLevel: logLevel,
		},
	}
}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
)

// log function with color support
func (l *loggerImpl) log(messageLevel LogLevel, levelLabel string, args ...interface{}) {
	if l.logLevel <= messageLevel {
		_, file, line, _ := runtime.Caller(2)
		shortFile := filepath.Base(file)

		color := getColorForLevel(messageLevel)
		str := fmt.Sprintf("%s%s [%s] %s:%d %v%s\n", color, time.Now().Format(time.RFC3339), levelLabel, shortFile, line, args, colorReset)

		fmt.Print(str)
	}
}

// getColorForLevel returns the color based on the log level
func getColorForLevel(level LogLevel) string {
	switch level {
	case LogLevelDebug:
		return colorBlue
	case LogLevelInfo:
		return colorGreen
	case LogLevelWarn:
		return colorYellow
	case LogLevelError:
		return colorRed
	default:
		return colorReset
	}
}

// GetLogLevelForEnvironment returns LogLevel based on the environment setting
func getLogLevelForEnvironment() LogLevel {
	switch os.Getenv("ENV") {
	case "production":
		return LogLevelError
	case "staging":
		return LogLevelWarn
	default:
		return LogLevelDebug
	}
}

// Debug logs a message at debug level
func (l *loggerImpl) Debug(args ...interface{}) {
	l.log(LogLevelDebug, "DEBUG", args...)
}

// Info logs a message at info level
func (l *loggerImpl) Info(args ...interface{}) {
	l.log(LogLevelInfo, "INFO", args...)
}

// Warn logs a message at warn level
func (l *loggerImpl) Warn(args ...interface{}) {
	l.log(LogLevelWarn, "WARN", args...)
}

// Error logs a message at error level
func (l *loggerImpl) Error(args ...interface{}) {
	l.log(LogLevelError, "ERROR", args...)
}
