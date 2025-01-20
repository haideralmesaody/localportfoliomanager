package utils

import "github.com/sirupsen/logrus"

// Logger wraps logrus.Logger
type Logger struct {
	*logrus.Logger
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.Errorf(format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.Infof(format, args...)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.Debugf(format, args...)
}

func NewLogger() *Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return &Logger{Logger: logger}
}
