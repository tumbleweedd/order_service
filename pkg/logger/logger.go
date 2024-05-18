package logger

import "context"

type Logger interface {
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)

	DebugContext(ctx context.Context, msg string, fields ...any)
	InfoContext(ctx context.Context, msg string, fields ...any)
	WarnContext(ctx context.Context, msg string, fields ...any)
	ErrorContext(ctx context.Context, msg string, fields ...any)
}

type Attr struct {
	Key   string
	Value any
}

func String(key, value string) Attr {
	return Attr{Key: key, Value: value}
}

func Int(key string, value int) Attr {
	return Attr{Key: key, Value: value}
}
