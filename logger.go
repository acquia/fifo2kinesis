package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
)

const (
	LOG_CRIT   int = 2
	LOG_ERROR  int = 3
	LOG_WARN   int = 4
	LOG_NOTICE int = 5
	LOG_INFO   int = 6
	LOG_DEBUG  int = 7
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
		CritLogger:   log.New(handler[2], "CRIT\t", log.Ldate|log.Ltime|log.LUTC),
		ErrorLogger:  log.New(handler[3], "ERROR\t", log.Ldate|log.Ltime|log.LUTC),
		WarnLogger:   log.New(handler[4], "WARN\t", log.Ldate|log.Ltime|log.LUTC),
		NoticeLogger: log.New(handler[5], "NOTICE\t", log.Ldate|log.Ltime|log.LUTC),
		InfoLogger:   log.New(handler[6], "INFO\t", log.Ldate|log.Ltime|log.LUTC),
		DebugLogger:  log.New(handler[7], "DEBUG\t", log.Ldate|log.Ltime|log.LUTC),
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
