package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

type Logger struct {
	level  Level
	format string
}

var defaultLogger = NewLogger(os.Getenv("APP_ENV"))

func Debug(msg string, args ...any) { defaultLogger.log(LevelDebug, msg, args...) }
func Info(msg string, args ...any)  { defaultLogger.log(LevelInfo, msg, args...) }
func Warn(msg string, args ...any)  { defaultLogger.log(LevelWarn, msg, args...) }
func Error(msg string, args ...any) { defaultLogger.log(LevelError, msg, args...) }

func Init(env string) {
	defaultLogger = NewLogger(env)
}

func NewLogger(env string) *Logger {
	level := LevelInfo
	format := "text"
	if strings.EqualFold(env, "production") {
		format = "json"
	}
	return &Logger{level: level, format: format}
}

func (l *Logger) log(level Level, msg string, args ...any) {
	_, file, line, _ := runtime.Caller(2)
	shortFile := file
	if idx := strings.LastIndex(file, "/"); idx != -1 {
		shortFile = file[idx+1:]
	}

	timestamp := time.Now().Format(time.RFC3339)
	content := fmt.Sprintf(msg, args...)

	if l.format == "json" {
		out := fmt.Sprintf(
			`{"time":"%s","level":"%s","msg":"%s","file":"%s:%d"}`,
			timestamp, level, escapeJSON(content), shortFile, line,
		)
		log.Println(out)
	} else {
		out := fmt.Sprintf("[%s] %-5s %s (%s:%d)", timestamp, level, content, shortFile, line)
		log.Println(out)
	}
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `"`, `'`)
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func (l *Logger) SetOutputFile(filename string) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	log.SetOutput(f)
	return nil
}
