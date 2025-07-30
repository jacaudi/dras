package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// Level represents logging levels
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

// Logger provides structured logging with levels
type Logger struct {
	level  Level
	output io.Writer
	logger *log.Logger
}

// New creates a new logger with the specified level
func New(level Level) *Logger {
	return &Logger{
		level:  level,
		output: os.Stdout,
		logger: log.New(os.Stdout, "", 0),
	}
}

// NewWithOutput creates a new logger with custom output
func NewWithOutput(level Level, output io.Writer) *Logger {
	return &Logger{
		level:  level,
		output: output,
		logger: log.New(output, "", 0),
	}
}

// SetLevel sets the minimum logging level
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// ParseLevel converts a string to a Level
func ParseLevel(level string) Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

// shouldLog checks if a message should be logged based on level
func (l *Logger) shouldLog(level Level) bool {
	return level >= l.level
}

// formatMessage formats a log message with timestamp and level
func (l *Logger) formatMessage(level Level, format string, args ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelName := levelNames[level]
	message := fmt.Sprintf(format, args...)
	return fmt.Sprintf("[%s] %s: %s", timestamp, levelName, message)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.shouldLog(DEBUG) {
		l.logger.Print(l.formatMessage(DEBUG, format, args...))
	}
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	if l.shouldLog(INFO) {
		l.logger.Print(l.formatMessage(INFO, format, args...))
	}
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.shouldLog(WARN) {
		l.logger.Print(l.formatMessage(WARN, format, args...))
	}
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	if l.shouldLog(ERROR) {
		l.logger.Print(l.formatMessage(ERROR, format, args...))
	}
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	if l.shouldLog(FATAL) {
		l.logger.Print(l.formatMessage(FATAL, format, args...))
		os.Exit(1)
	}
}

// WithField adds context to log messages
func (l *Logger) WithField(key, value string) *FieldLogger {
	return &FieldLogger{
		logger: l,
		fields: map[string]string{key: value},
	}
}

// WithFields adds multiple context fields to log messages
func (l *Logger) WithFields(fields map[string]string) *FieldLogger {
	return &FieldLogger{
		logger: l,
		fields: fields,
	}
}

// FieldLogger provides contextual logging
type FieldLogger struct {
	logger *Logger
	fields map[string]string
}

// formatWithFields formats a message with context fields
func (fl *FieldLogger) formatWithFields(format string, args ...interface{}) string {
	message := fmt.Sprintf(format, args...)
	if len(fl.fields) == 0 {
		return message
	}
	
	var fieldStrings []string
	for key, value := range fl.fields {
		fieldStrings = append(fieldStrings, fmt.Sprintf("%s=%s", key, value))
	}
	return fmt.Sprintf("%s [%s]", message, strings.Join(fieldStrings, ", "))
}

// Debug logs a debug message with fields
func (fl *FieldLogger) Debug(format string, args ...interface{}) {
	if fl.logger.shouldLog(DEBUG) {
		message := fl.formatWithFields(format, args...)
		fl.logger.logger.Print(fl.logger.formatMessage(DEBUG, message))
	}
}

// Info logs an info message with fields
func (fl *FieldLogger) Info(format string, args ...interface{}) {
	if fl.logger.shouldLog(INFO) {
		message := fl.formatWithFields(format, args...)
		fl.logger.logger.Print(fl.logger.formatMessage(INFO, message))
	}
}

// Warn logs a warning message with fields
func (fl *FieldLogger) Warn(format string, args ...interface{}) {
	if fl.logger.shouldLog(WARN) {
		message := fl.formatWithFields(format, args...)
		fl.logger.logger.Print(fl.logger.formatMessage(WARN, message))
	}
}

// Error logs an error message with fields
func (fl *FieldLogger) Error(format string, args ...interface{}) {
	if fl.logger.shouldLog(ERROR) {
		message := fl.formatWithFields(format, args...)
		fl.logger.logger.Print(fl.logger.formatMessage(ERROR, message))
	}
}

// Fatal logs a fatal message with fields and exits
func (fl *FieldLogger) Fatal(format string, args ...interface{}) {
	if fl.logger.shouldLog(FATAL) {
		message := fl.formatWithFields(format, args...)
		fl.logger.logger.Print(fl.logger.formatMessage(FATAL, message))
		os.Exit(1)
	}
}

// Default logger instance
var defaultLogger = New(INFO)

// SetDefaultLevel sets the level for the default logger
func SetDefaultLevel(level Level) {
	defaultLogger.SetLevel(level)
}

// Package-level convenience functions using the default logger
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

func WithField(key, value string) *FieldLogger {
	return defaultLogger.WithField(key, value)
}

func WithFields(fields map[string]string) *FieldLogger {
	return defaultLogger.WithFields(fields)
}