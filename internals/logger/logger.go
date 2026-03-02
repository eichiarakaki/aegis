package logger

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	reset  = "\033[0m"
	gray   = "\033[90m"
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	cyan   = "\033[36m"
)

type Logger struct {
	requestID string
	sessionID string
	component string
	fields    map[string]string
}

var mu sync.Mutex

// -------- Constructors --------

func New() *Logger {
	return &Logger{
		fields: make(map[string]string),
	}
}

func WithRequestID(id string) *Logger {
	return New().WithRequestID(id)
}

func WithSessionID(id string) *Logger {
	return New().WithSessionID(id)
}

func WithComponent(name string) *Logger {
	return New().WithComponent(name)
}

func (l *Logger) WithRequestID(id string) *Logger {
	l.requestID = id
	return l
}

func (l *Logger) WithSessionID(id string) *Logger {
	l.sessionID = id
	return l
}

func (l *Logger) WithComponent(name string) *Logger {
	l.component = name
	return l
}

func (l *Logger) WithField(key, value string) *Logger {
	l.fields[key] = value
	return l
}

// -------- Core --------

func timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func (l *Logger) buildContext() string {
	var parts []string

	if l.requestID != "" {
		parts = append(parts, "req="+l.requestID)
	}
	if l.sessionID != "" {
		parts = append(parts, "sess="+l.sessionID)
	}
	if l.component != "" {
		parts = append(parts, "comp="+l.component)
	}
	for k, v := range l.fields {
		parts = append(parts, k+"="+v)
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ") + " "
}

func (l *Logger) log(level, color string, msg string) {
	mu.Lock()
	defer mu.Unlock()

	prefix := fmt.Sprintf(
		"%s%s%s %s[%s]%s ",
		gray, timestamp(), reset,
		color, level, reset,
	)

	context := l.buildContext()

	fmt.Println(prefix + context + msg)
}

// -------- Level Methods --------

func (l *Logger) Info(msg string) {
	l.log("INFO", green, msg)
}

func (l *Logger) Infof(format string, args ...any) {
	l.log("INFO", green, fmt.Sprintf(format, args...))
}

func (l *Logger) Warn(msg string) {
	l.log("WARN", yellow, msg)
}

func (l *Logger) Warnf(format string, args ...any) {
	l.log("WARN", yellow, fmt.Sprintf(format, args...))
}

func (l *Logger) Error(msg string) {
	l.log("ERROR", red, msg)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.log("ERROR", red, fmt.Sprintf(format, args...))
}

func (l *Logger) Debug(msg string) {
	l.log("DEBUG", cyan, msg)
}

func (l *Logger) Debugf(format string, args ...any) {
	l.log("DEBUG", cyan, fmt.Sprintf(format, args...))
}

// -------- Global Simple API (Backwards Compatible) --------

func Info(args ...any) {
	New().Info(fmt.Sprint(args...))
}

func Infof(format string, args ...any) {
	New().Infof(format, args...)
}

func Warn(args ...any) {
	New().Warn(fmt.Sprint(args...))
}

func Warnf(format string, args ...any) {
	New().Warnf(format, args...)
}

func Error(args ...any) {
	New().Error(fmt.Sprint(args...))
}

func Errorf(format string, args ...any) {
	New().Errorf(format, args...)
}

func Debug(args ...any) {
	New().Debug(fmt.Sprint(args...))
}

func Debugf(format string, args ...any) {
	New().Debugf(format, args...)
}
