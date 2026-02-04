package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

var defaultLogger *slog.Logger

func init() {
	defaultLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func InitLogger(verbose, debug bool) {
	var level slog.Level
	if debug {
		level = slog.LevelDebug
	} else if verbose {
		level = slog.LevelInfo
	} else {
		level = slog.LevelWarn
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     level,
		AddSource: debug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				return redactSensitiveData(a)
			}
			return a
		},
	})

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

func SetOutput(w io.Writer) {
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

var sensitiveKeys = []string{
	"password",
	"passwd",
	"secret",
	"token",
	"api_key",
	"apikey",
	"access_key",
	"private_key",
	"credential",
	"auth",
}

func redactSensitiveData(attr slog.Attr) slog.Attr {
	msg := attr.Value.String()
	lower := strings.ToLower(msg)

	for _, key := range sensitiveKeys {
		if strings.Contains(lower, key) {
			msg = redactValue(msg, key)
		}
	}

	return slog.String(attr.Key, msg)
}

func redactValue(text, keyword string) string {
	return strings.ReplaceAll(
		strings.ReplaceAll(text, keyword+"=", keyword+"=[REDACTED]"),
		keyword+": ",
		keyword+": [REDACTED]",
	)
}
