package http

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/kdv2001/onlyMetrics/internal/pkg/logger"
)

// AddLoggerToContextMiddleware помещает logger в context
func AddLoggerToContextMiddleware(sugarLogger *zap.SugaredLogger) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = logger.ToContext(ctx, sugarLogger)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

// RequestMiddleware middleware для логирования запросов
func RequestMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			defer func() {
				logger.Infof(r.Context(), "request: url: %s; method: %s; processing time: %s",
					r.URL.String(), r.Method, time.Since(start).String())
			}()

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// ResponseMiddleware middleware для логирования ответов
func ResponseMiddleware() func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			updatedWriter := NewWriterWithLogging(w)
			defer func() {
				defer func() {
					logger.Infof(r.Context(), "response: status code: %d, datasize: %d bytes",
						updatedWriter.statusCode,
						updatedWriter.responseSize)
				}()
			}()

			next.ServeHTTP(updatedWriter, r)
		}

		return http.HandlerFunc(fn)
	}
}

// WriterWithLogging реализация интерфейса writer для перехвата информации ответа
type WriterWithLogging struct {
	statusCode   int
	responseSize int

	baseWriter http.ResponseWriter
}

// NewWriterWithLogging создание нового WriterWithLogging объекта
func NewWriterWithLogging(baseWriter http.ResponseWriter) *WriterWithLogging {
	return &WriterWithLogging{
		baseWriter: baseWriter,
	}
}

// Write ...
func (w *WriterWithLogging) Write(b []byte) (int, error) {
	w.responseSize = len(b)
	return w.baseWriter.Write(b)
}

// WriteHeader ...
func (w *WriterWithLogging) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.baseWriter.WriteHeader(statusCode)
}

// Header ...
func (w *WriterWithLogging) Header() http.Header {
	return w.baseWriter.Header()
}
