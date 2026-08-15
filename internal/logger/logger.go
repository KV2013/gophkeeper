package logger

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level string) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, err
	}

	// cfg := zap.NewProductionConfig()
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = lvl
	return cfg.Build()
}

type ZapLogFormatter struct {
	Logger    *zap.Logger
	UserIDKey any
}

func (f *ZapLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {

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
	}
	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		fields = append(fields, zap.String("request_id", reqID))
	}
	if f.UserIDKey != nil {
		if uid, ok := r.Context().Value(f.UserIDKey).(string); ok && uid != "" {
			fields = append(fields, zap.String("user_id", uid))
		}
	}

	return &ZapLogEntry{
		logger: f.Logger.With(fields...),
	}
}

type ZapLogEntry struct {
	logger *zap.Logger
}

func (e *ZapLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	fields := []zap.Field{
		zap.Int("status", status),
		zap.Int("bytes", bytes),
		zap.Duration("latency", elapsed),
	}
	lvl := statusLevel(status)
	e.logger.Log(lvl, "", fields...)
}

func (e *ZapLogEntry) Panic(v interface{}, stack []byte) {
	e.logger.Error("request panic",
		zap.Any("panic", v),
		zap.ByteString("stack", stack),
	)
}

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
