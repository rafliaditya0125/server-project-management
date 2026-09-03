package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
)

const (
	ColorRed    = "\033[0;31m"
	ColorGreen  = "\033[0;32m"
	ColorYellow = "\033[1;33m"
	ColorBlue   = "\033[0;34m"
	ColorCyan   = "\033[0;36m"
	ColorBold   = "\033[1m"
	ColorReset  = "\033[0m"
)

type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	errOut io.Writer
}

var DefaultLogger = NewLogger(os.Stdout, os.Stderr)

func NewLogger(out, errOut io.Writer) *Logger {
	return &Logger{
		out:    out,
		errOut: errOut,
	}
}

func (l *Logger) Info(format string, a ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(l.out, "%s[INFO]%s %s\n", ColorBlue, ColorReset, msg)
}

func (l *Logger) Success(format string, a ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(l.out, "%s[SUCCESS]%s %s\n", ColorGreen, ColorReset, msg)
}

func (l *Logger) Warn(format string, a ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(l.out, "%s[WARN]%s %s\n", ColorYellow, ColorReset, msg)
}

func (l *Logger) Error(format string, a ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(l.errOut, "%s[ERROR]%s %s\n", ColorRed, ColorReset, msg)
}

func (l *Logger) Raw(format string, a ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.out, format, a...)
}

func Info(format string, a ...any) {
	DefaultLogger.Info(format, a...)
}

func Success(format string, a ...any) {
	DefaultLogger.Success(format, a...)
}

func Warn(format string, a ...any) {
	DefaultLogger.Warn(format, a...)
}

func Error(format string, a ...any) {
	DefaultLogger.Error(format, a...)
}

func Raw(format string, a ...any) {
	DefaultLogger.Raw(format, a...)
}
