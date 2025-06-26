package http

import (
	"compress/gzip"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/kdv2001/onlyMetrics/internal/pkg/logger"
)

// defaultAcceptedEncodingTypes поддерживаемы типы для компрессии
var defaultAcceptedEncodingTypes = map[string]struct{}{
	TextHTML:        {},
	ApplicationJSON: {},
}

// GetDefaultAcceptedEncodingData возвращает стандартный типа для компрессии
func GetDefaultAcceptedEncodingData() map[string]struct{} {
	return defaultAcceptedEncodingTypes
}

// compressWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки
type compressWriter struct {
	w              http.ResponseWriter
	compressWriter io.WriteCloser
}

func newCompressWriter(w http.ResponseWriter, h http.Header, acceptedEncodingData map[string]struct{}) *compressWriter {
	values := h.Values(Accept)
	accepted := false
	for _, v := range values {
		if _, isExist := acceptedEncodingData[v]; isExist {
			accepted = true
			break
		}
	}

	if !accepted {
		return &compressWriter{
			w: w,
		}
	}

	compressAlg := ""
	if acceptEncodingValues := h.Values(AcceptEncoding); len(acceptEncodingValues) > 0 {
		compressAlg = acceptEncodingValues[0]
	}

	switch compressAlg {
	case Gzip:
		w.Header().Set(ContentEncoding, Gzip)
		cw := gzip.NewWriter(w)
		return &compressWriter{
			w:              w,
			compressWriter: cw,
		}
	}

	return &compressWriter{
		w: w,
	}
}

// Header ...
func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

// Write ...
func (c *compressWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	if c.compressWriter != nil {
		return c.compressWriter.Write(p)
	}

	return c.w.Write(p)
}

// WriteHeader ...
func (c *compressWriter) WriteHeader(statusCode int) {
	c.w.WriteHeader(statusCode)
}

// Close ...
func (c *compressWriter) Close() error {
	if c.compressWriter == nil {
		return nil
	}

	return c.compressWriter.Close()
}

// CompressMiddleware создаёт middleware для сжатия данных
// TODO можно переделать на опции
func CompressMiddleware(encodingTypes map[string]struct{}) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ow := newCompressWriter(w, r.Header, encodingTypes)
			defer ow.Close()
			next.ServeHTTP(ow, r)
		}

		return http.HandlerFunc(fn)
	}
}

// compressReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// DecompressMiddleware создаёт middleware для декомпрессии
func DecompressMiddleware() func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			val := r.Header.Get(ContentEncoding)
			var err error
			var zr io.ReadCloser
			switch val {
			case Gzip:
				zr, err = gzip.NewReader(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			// случай пустого заголовка
			case "":
				zr = r.Body
			default:
				http.Error(w, "error: unsupported Content-Encoding ", http.StatusBadRequest)
				return
			}

			// меняем тело запроса на новое
			r.Body = zr
			defer zr.Close()

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

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
