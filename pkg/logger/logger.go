// Package logger предоставляет функции для работы с логером:
// - логирование сообщений с различным уровнем;
// - помещение и извлечение логгера из контекста.
package logger

import (
	"context"

	"go.uber.org/zap"
)

// loggerKey приватный тип, необходим для извлечения логгера
// из контекста только в рамках этого пакета
type loggerKey string

const key loggerKey = "contextLogger"

// ToContext помещает logger в контекст
func ToContext(ctx context.Context, sugarLogger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, key, sugarLogger)
}

// Infof логгирует сообщение с уровнем INFO
func Infof(ctx context.Context, format string, args ...any) {
	logger := FromContext(ctx)
	logger.Infof(format, args...)
}

// Errorf логгирует сообщение с уровнем ERROR
func Errorf(ctx context.Context, format string, args ...any) {
	logger := FromContext(ctx)
	logger.Errorf(format, args...)
}

var defaultSugarLogger, _ = zap.NewDevelopment()

// FromContext извлекает logger из контекста
func FromContext(ctx context.Context) *zap.SugaredLogger {
	logger := ctx.Value(key)
	sugarLogger, ok := logger.(*zap.SugaredLogger)
	if !ok {
		return defaultSugarLogger.Sugar()
	}

	return sugarLogger
}
