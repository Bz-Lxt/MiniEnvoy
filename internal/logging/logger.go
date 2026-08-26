package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

func New(level string) *slog.Logger {
	var lv slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lv = slog.LevelDebug
	case "warn", "warning":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{
		Level: lv,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(interface{ Time() interface{} }); ok {
					_ = t
				}
			}
			return a
		},
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func Discard() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
