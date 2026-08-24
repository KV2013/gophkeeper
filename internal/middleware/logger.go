package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// statusRecorder — обёртка ResponseWriter, захватывающая код статуса и
// количество записанных байтов.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

// WriteHeader запоминает первый записанный код статуса.
func (w *statusRecorder) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

// Write учитывает записанные байты.
func (w *statusRecorder) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// ZapLogger — middleware, логирующий HTTP-запросы с помощью zap.
func ZapLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}

			next.ServeHTTP(rec, r)

			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			fields := []zap.Field{
				zap.String("method", r.Method),
				zap.String("uri", r.RequestURI),
				zap.String("url", scheme+"://"+r.Host+r.RequestURI),
				zap.String("proto", r.Proto),
				zap.String("remote_addr", r.RemoteAddr),
				zap.Int("status", rec.status),
				zap.Int("bytes", rec.bytes),
				zap.Duration("latency", time.Since(start)),
			}
			if reqID := RequestIDFromContext(r.Context()); reqID != "" {
				fields = append(fields, zap.String("request_id", reqID))
			}
			if uid := UserID(r.Context()); uid != "" {
				fields = append(fields, zap.String("user_id", uid))
			}

			logger.Log(statusLevel(rec.status), "", fields...)
		})
	}
}

// statusLevel определяет уровень логирования по коду статуса.
func statusLevel(status int) zapcore.Level {
	switch {
	case status >= 500:
		return zap.ErrorLevel
	case status >= 400:
		return zap.WarnLevel
	default:
		return zap.InfoLevel
	}
}
