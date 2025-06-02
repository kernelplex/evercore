package main

import (
    "log"
    "os"
)

type Logger struct {
    verbose bool
    log     *log.Logger
}

func NewLogger(verbose bool) *Logger {
    return &Logger{
        verbose: verbose,
        log:     log.New(os.Stderr, "", log.LstdFlags),
    }
}

func (l *Logger) Debug(format string, v ...interface{}) {
    if l.verbose {
        l.log.Printf("[DEBUG] "+format, v...)
    }
}

func (l *Logger) Info(format string, v ...interface{}) {
    l.log.Printf("[INFO] "+format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
    l.log.Printf("[ERROR] "+format, v...)
}
