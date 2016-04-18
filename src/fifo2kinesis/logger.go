package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
)

const (
	LOG_NONE = iota
	LOG_CRIT
	LOG_ERROR
	LOG_WARN
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

type Logger struct {
	CritLogger   *log.Logger
	ErrorLogger  *log.Logger
	WarnLogger   *log.Logger
	NoticeLogger *log.Logger
	InfoLogger   *log.Logger
	DebugLogger  *log.Logger
}

func NewLogger(level int) *Logger {

	handler := make([]io.Writer, 8)
	for k, _ := range handler {
		if k <= level {
			handler[k] = os.Stdout
		} else {
			handler[k] = ioutil.Discard
		}
	}

	return &Logger{
		CritLogger:   log.New(handler[LOG_CRIT], "CRIT\t", log.Ldate|log.Ltime|log.LUTC),
		ErrorLogger:  log.New(handler[LOG_ERROR], "ERROR\t", log.Ldate|log.Ltime|log.LUTC),
		WarnLogger:   log.New(handler[LOG_WARN], "WARN\t", log.Ldate|log.Ltime|log.LUTC),
		NoticeLogger: log.New(handler[LOG_NOTICE], "NOTICE\t", log.Ldate|log.Ltime|log.LUTC),
		InfoLogger:   log.New(handler[LOG_INFO], "INFO\t", log.Ldate|log.Ltime|log.LUTC),
		DebugLogger:  log.New(handler[LOG_DEBUG], "DEBUG\t", log.Ldate|log.Ltime|log.LUTC),
	}
}

func (l *Logger) Crit(format string, v ...interface{}) {
	l.CritLogger.Printf(format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.ErrorLogger.Printf(format, v...)
}

func (l *Logger) Warn(format string, v ...interface{}) {
	l.WarnLogger.Printf(format, v...)
}

func (l *Logger) Notice(format string, v ...interface{}) {
	l.NoticeLogger.Printf(format, v...)
}

func (l *Logger) Info(format string, v ...interface{}) {
	l.InfoLogger.Printf(format, v...)
}

func (l *Logger) Debug(format string, v ...interface{}) {
	l.DebugLogger.Printf(format, v...)
}

func (l *Logger) Fatal(v ...interface{}) {
	l.CritLogger.Fatal(v...)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.CritLogger.Fatalf(format, v...)
}

func (l *Logger) Panic(v ...interface{}) {
	l.CritLogger.Panic(v...)
}

func (l *Logger) Panicf(format string, v ...interface{}) {
	l.CritLogger.Panicf(format, v...)
}
