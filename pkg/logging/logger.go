package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"blog-api/pkg/settings"
)

// config
type LoggerConfig struct {
	Level string
}

func (c *LoggerConfig) Setup() []settings.EnvLoadable {
	return []settings.EnvLoadable{
		settings.Item[string]{Name: "LOG_LEVEL", Default: "INFO", Field: &c.Level},
	}
}

// logger
var slogLogger *slog.Logger

type printfLogger struct{}

func Init(cfg *LoggerConfig) {
	var lvl slog.Level
	switch cfg.Level {
	case "DEBUG":
		lvl = slog.LevelDebug
	case "INFO":
		lvl = slog.LevelInfo
	case "WARN":
		lvl = slog.LevelWarn
	case "ERROR":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handlerOpts := &slog.HandlerOptions{Level: lvl}
	slogLogger = slog.New(slog.NewJSONHandler(os.Stdout, handlerOpts))
}

func L() *printfLogger {
	return &printfLogger{}
}

// printf-style methods
func (l *printfLogger) Debug(format string, args ...any) {
	slogLogger.Log(context.Background(), slog.LevelDebug, fmt.Sprintf(format, args...))
}

func (l *printfLogger) Info(format string, args ...any) {
	slogLogger.Log(context.Background(), slog.LevelInfo, fmt.Sprintf(format, args...))
}

func (l *printfLogger) Warn(format string, args ...any) {
	slogLogger.Log(context.Background(), slog.LevelWarn, fmt.Sprintf(format, args...))
}

func (l *printfLogger) Error(format string, args ...any) {
	slogLogger.Log(context.Background(), slog.LevelError, fmt.Sprintf(format, args...))
}

// returns the underlying slog.Logger
func Raw() *slog.Logger { return slogLogger }
