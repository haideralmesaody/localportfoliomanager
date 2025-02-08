package utils

import (
	"log"
	"os"
)

type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

type AppLogger struct {
	debug *log.Logger
	info  *log.Logger
	error *log.Logger
}

func NewAppLogger() *AppLogger {
	return &AppLogger{
		debug: log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile),
		info:  log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime),
		error: log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (l *AppLogger) Debug(msg string, args ...interface{}) {
	l.debug.Printf(msg, args...)
}

func (l *AppLogger) Info(msg string, args ...interface{}) {
	l.info.Printf(msg, args...)
}

func (l *AppLogger) Error(msg string, args ...interface{}) {
	l.error.Printf(msg, args...)
}
