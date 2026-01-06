package x

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	LevelDebug = -4
	LevelInfo  = 0
	LevelWarn  = 4
	LevelError = 8
)

var DefaultLogger = NewLogger(
	LevelInfo,
	time.UTC,
	"2006.01.02 15:04:05 (MST)",
	nil,
)

type LogCallback func(ts, level, src string, args ...any)
type Logger struct {
	level    int
	timezone *time.Location
	format   string
	callback LogCallback
}

func NewLogger(
	level int, tz *time.Location, format string, cb LogCallback,
) *Logger {
	return &Logger{
		level:    level,
		timezone: tz,
		format:   format,
		callback: cb,
	}
}

func (l *Logger) SetLevel(level int) { l.level = level }
func (l *Logger) SetTimezone(name string) error {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return err
	}
	l.timezone = loc
	return nil
}
func (l *Logger) SetFormat(format string) { l.format = format }
func (l *Logger) SetCallback(cb LogCallback) {
	l.callback = cb
}

func (l *Logger) GetLevel() string {
	switch l.level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", l.level)
	}
}
func (l *Logger) GetTimezone() *time.Location { return l.timezone }
func (l *Logger) GetFormat() string           { return l.format }

func (l *Logger) Debug(args ...any) {
	if LevelDebug < l.level {
		return
	}
	l.output("DEBUG", args...)
}

func (l *Logger) Info(args ...any) {
	if LevelInfo < l.level {
		return
	}
	l.output("INFO", args...)
}

func (l *Logger) Warn(args ...any) {
	if LevelWarn < l.level {
		return
	}
	l.output("WARN", args...)
}

func (l *Logger) Error(args ...any) {
	if LevelError < l.level {
		return
	}
	l.output("ERROR", args...)
}

func (l *Logger) output(level string, args ...any) {
	ts := time.Now().In(l.timezone).Format(l.format)
	_, file, line, ok := runtime.Caller(2)
	src := "unknown"
	if ok {
		src = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	if l.callback != nil {
		l.callback(ts, level, src, args...)
	} else {
		strArgs := make([]string, len(args))
		for i, a := range args {
			strArgs[i] = fmt.Sprint(a) // any → string 변환
		}
		line := strings.Join(strArgs, " ")
		fmt.Printf("%s %-5s %s %s\n", ts, level, line, src)
	}
}
