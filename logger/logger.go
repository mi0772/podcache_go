package logger

import (
	"fmt"
	"os"
	"time"
)

type Level int

const (
	LOG_TRACE Level = iota
	LOG_DEBUG
	LOG_INFO
	LOG_WARN
	LOG_ERROR
)

var LogLevel Level = LOG_INFO
var p = fmt.Printf

func Write(level Level, format string, values ...any) {
	if level < LogLevel {
		return
	}
	go func(t time.Time) {
		if len(values) > 0 {
			p("%d-%d-%d %d:%d:%d - %s\n", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), format, values)
		} else {
			p("%d-%d-%d %d:%d:%d - %s\n", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), format)
		}
	}(time.Now())
}

func WriteInfo(format string, values ...any) {
	Write(LOG_INFO, format, values...)
}
func WriteDebug(format string, values ...any) {
	Write(LOG_DEBUG, format, values...)
}
func WriteTrace(format string, values ...any) {
	Write(LOG_TRACE, format, values...)
}
func WriteWarn(format string, values ...any) {
	Write(LOG_WARN, format, values...)
}
func WriteError(format string, values ...any) {
	Write(LOG_ERROR, format, values...)
}
func WriteFatal(format string, values ...any) {
	WriteError(format, values...)
	os.Exit(1)
}
