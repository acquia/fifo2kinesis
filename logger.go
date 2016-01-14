package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
)

const (
	LOG_EMERG  int = 0
	LOG_ALERT  int = 1
	LOG_CRIT   int = 2
	LOG_ERROR  int = 3
	LOG_WARN   int = 4
	LOG_NOTICE int = 5
	LOG_INFO   int = 6
	LOG_DEBUG  int = 7
)

type Logger struct {
	EmergLogger  *log.Logger
	AlertLogger  *log.Logger
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
		EmergLogger:  log.New(handler[0], "EMERG\t", log.Ldate|log.Ltime),
		AlertLogger:  log.New(handler[1], "ALERT\t", log.Ldate|log.Ltime),
		CritLogger:   log.New(handler[2], "CRIT\t", log.Ldate|log.Ltime),
		ErrorLogger:  log.New(handler[3], "ERROR\t", log.Ldate|log.Ltime),
		WarnLogger:   log.New(handler[4], "WARN\t", log.Ldate|log.Ltime),
		NoticeLogger: log.New(handler[5], "NOTICE\t", log.Ldate|log.Ltime),
		InfoLogger:   log.New(handler[6], "INFO\t", log.Ldate|log.Ltime),
		DebugLogger:  log.New(handler[7], "DEBUG\t", log.Ldate|log.Ltime),
	}
}

func (l *Logger) Emerg(format string, v ...interface{}) {
	l.EmergLogger.Printf(format, v...)
}

func (l *Logger) Alert(format string, v ...interface{}) {
	l.AlertLogger.Printf(format, v...)
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
	l.EmergLogger.Fatal(v...)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.EmergLogger.Fatalf(format, v...)
}

func (l *Logger) Panic(v ...interface{}) {
	l.EmergLogger.Panic(v...)
}

func (l *Logger) Panicf(format string, v ...interface{}) {
	l.EmergLogger.Panicf(format, v...)
}
