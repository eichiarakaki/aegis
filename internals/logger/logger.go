package logger

import (
	"fmt"
	"os"
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

// debugEnabled is set once at startup from AEGIS_LOG_LEVEL=debug.
var debugEnabled = func() bool {
	return strings.ToLower(os.Getenv("AEGIS_LOG_LEVEL")) == "debug"
}()

type Logger struct {
	requestID string
	sessionID string
	component string
	fields    map[string]string
}

var mu sync.Mutex

// -------- Constructors --------

func New() *Logger {
	return &Logger{fields: make(map[string]string)}
}

func WithRequestID(id string) *Logger   { return New().WithRequestID(id) }
func WithSessionID(id string) *Logger   { return New().WithSessionID(id) }
func WithComponent(name string) *Logger { return New().WithComponent(name) }

func (l *Logger) WithRequestID(id string) *Logger     { l.requestID = id; return l }
func (l *Logger) WithSessionID(id string) *Logger     { l.sessionID = id; return l }
func (l *Logger) WithComponent(name string) *Logger   { l.component = name; return l }
func (l *Logger) WithField(key, value string) *Logger { l.fields[key] = value; return l }

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

func (l *Logger) log(level, color, msg string) {
	mu.Lock()
	defer mu.Unlock()

	prefix := fmt.Sprintf("%s%s%s %s[%s]%s ",
		gray, timestamp(), reset,
		color, level, reset,
	)
	fmt.Println(prefix + l.buildContext() + msg)
}

// -------- Level Methods --------

func (l *Logger) Info(msg string)           { l.log("INFO", green, msg) }
func (l *Logger) Infof(f string, a ...any)  { l.log("INFO", green, fmt.Sprintf(f, a...)) }
func (l *Logger) Warn(msg string)           { l.log("WARN", yellow, msg) }
func (l *Logger) Warnf(f string, a ...any)  { l.log("WARN", yellow, fmt.Sprintf(f, a...)) }
func (l *Logger) Error(msg string)          { l.log("ERROR", red, msg) }
func (l *Logger) Errorf(f string, a ...any) { l.log("ERROR", red, fmt.Sprintf(f, a...)) }

func (l *Logger) Debug(msg string) {
	if !debugEnabled {
		return
	}
	l.log("DEBUG", cyan, msg)
}

func (l *Logger) Debugf(f string, a ...any) {
	if !debugEnabled {
		return
	}
	l.log("DEBUG", cyan, fmt.Sprintf(f, a...))
}

// -------- Global API --------

func Info(args ...any)          { New().Info(fmt.Sprint(args...)) }
func Infof(f string, a ...any)  { New().Infof(f, a...) }
func Warn(args ...any)          { New().Warn(fmt.Sprint(args...)) }
func Warnf(f string, a ...any)  { New().Warnf(f, a...) }
func Error(args ...any)         { New().Error(fmt.Sprint(args...)) }
func Errorf(f string, a ...any) { New().Errorf(f, a...) }

func Debug(args ...any) {
	if !debugEnabled {
		return
	}
	New().Debug(fmt.Sprint(args...))
}

func Debugf(f string, a ...any) {
	if !debugEnabled {
		return
	}
	New().Debugf(f, a...)
}
