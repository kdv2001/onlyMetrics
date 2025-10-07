package http

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/kdv2001/onlyMetrics/pkg/logger"
)

// hashWriter реализация интерфейса writer для перехвата информации ответа, последующего вычисления хэша
// и добавления вычисленного значения в заголовки.
type hashWriter struct {
	key string
	w   http.ResponseWriter
}

// NewHashWriter ...
func NewHashWriter(w http.ResponseWriter, key string) *hashWriter {
	return &hashWriter{
		key: key,
		w:   w,
	}
}

// Header ...
func (c *hashWriter) Header() http.Header {
	return c.w.Header()
}

// Write ...
func (c *hashWriter) Write(p []byte) (int, error) {
	hh := hmac.New(sha256.New, []byte(c.key))
	if _, err := hh.Write(p); err != nil {
		return c.w.Write(p)
	}

	bufSHA := hh.Sum(nil)

	str := hex.EncodeToString(bufSHA)
	c.Header().Set(HashSHA256, str)

	return c.w.Write(p)
}

// WriteHeader ...
func (c *hashWriter) WriteHeader(statusCode int) {
	c.w.WriteHeader(statusCode)
}

// NewSha256Middleware создаёт middleware для добавления хэш суммы.
func NewSha256Middleware(key string) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		f := func(w http.ResponseWriter, r *http.Request) {
			hashHeader := r.Header.Get(HashSHA256)
			if hashHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			hW := NewHashWriter(w, key)

			hh := hmac.New(sha256.New, []byte(key))
			body, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Errorf(r.Context(), "eroror read body: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			defer r.Body.Close()
			r.Body = io.NopCloser(bytes.NewBuffer(body))

			if _, err = hh.Write(body); err != nil {
				logger.Errorf(r.Context(), "eroror write body: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			hashSumReq, err := hex.DecodeString(hashHeader)
			if err != nil {
				logger.Errorf(r.Context(), "eroror decode header: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			hashSum := hh.Sum(nil)
			if equal := hmac.Equal(hashSum, hashSumReq); !equal {
				http.Error(w, "error compare request hash", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(hW, r)
		}

		return http.HandlerFunc(f)
	}
}

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
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки.
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

// CompressMiddleware создаёт middleware для сжатия данных.
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

// DecompressMiddleware создаёт middleware для декомпрессии.
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

// AddLoggerToContextMiddleware middleware для помещения logger в context.
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

// RequestMiddleware middleware для логирования запросов.
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

// ResponseMiddleware middleware для логирования ответов.
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

// WriterWithLogging реализация интерфейса writer для перехвата информации ответа и последующего его логгирования.
type WriterWithLogging struct {
	statusCode   int
	responseSize int

	baseWriter http.ResponseWriter
}

// NewWriterWithLogging создание нового WriterWithLogging объекта.
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
