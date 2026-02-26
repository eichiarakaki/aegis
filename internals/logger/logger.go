package logger

import (
	"fmt"
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

func timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func logPrint(level, color string, args ...any) {
	prefix := fmt.Sprintf(
		"%s%s%s %s[%s]%s ",
		gray, timestamp(), reset,
		color, level, reset,
	)

	fmt.Print(prefix)
	fmt.Println(args...)
}

func logPrintf(level, color, format string, args ...any) {
	prefix := fmt.Sprintf(
		"%s%s%s %s[%s]%s ",
		gray, timestamp(), reset,
		color, level, reset,
	)

	fmt.Print(prefix)
	fmt.Printf(format+"\n", args...)
}

func Info(args ...any) {
	logPrint("INFO", green, args...)
}

func Infof(format string, args ...any) {
	logPrintf("INFO", green, format, args...)
}

func Warn(args ...any) {
	logPrint("WARN", yellow, args...)
}

func Warnf(format string, args ...any) {
	logPrintf("WARN", yellow, format, args...)
}

func Error(args ...any) {
	logPrint("ERROR", red, args...)
}

func Errorf(format string, args ...any) {
	logPrintf("ERROR", red, format, args...)
}

func Debug(args ...any) {
	logPrint("DEBUG", cyan, args...)
}

func Debugf(format string, args ...any) {
	logPrintf("DEBUG", cyan, format, args...)
}
