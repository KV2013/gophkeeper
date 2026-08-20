package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(io.Discard)
	},
}

var gzipReaderPool = sync.Pool{
	New: func() interface{} {
		return new(gzip.Reader)
	},
}

func acquireGzipWriter(w io.Writer) *gzip.Writer {
	gw := gzipWriterPool.Get().(*gzip.Writer)
	gw.Reset(w)
	return gw
}

func releaseGzipWriter(gw *gzip.Writer) {
	gzipWriterPool.Put(gw)
}

func acquireGzipReader(r io.Reader) (*gzip.Reader, error) {
	zr := gzipReaderPool.Get().(*gzip.Reader)
	if err := zr.Reset(r); err != nil {
		gzipReaderPool.Put(zr)
		return nil, err
	}
	return zr, nil
}

func releaseGzipReader(zr *gzip.Reader) {
	gzipReaderPool.Put(zr)
}

func GzipCompression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ow := w

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")

		contentType := r.Header.Get("Content-Type")
		gzippableContentType := strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/html")

		if supportsGzip && gzippableContentType {
			gzw := NewGzipCompressWriter(w)
			ow = gzw
			defer gzw.Close()
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		isRequestGzipped := strings.Contains(contentEncoding, "gzip")
		if isRequestGzipped {
			cr, err := NewGzipCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		next.ServeHTTP(ow, r)
	})
}

type gzipCompressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func NewGzipCompressWriter(w http.ResponseWriter) *gzipCompressWriter {
	return &gzipCompressWriter{
		w:  w,
		zw: acquireGzipWriter(w),
	}
}

func (gz *gzipCompressWriter) Write(p []byte) (int, error) {
	return gz.zw.Write(p)
}
func (gz *gzipCompressWriter) Header() http.Header {
	return gz.w.Header()
}
func (gz *gzipCompressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		gz.w.Header().Set("Content-Encoding", "gzip")
	}
	gz.w.WriteHeader(statusCode)
}
func (gz *gzipCompressWriter) Close() error {
	err := gz.zw.Close()
	releaseGzipWriter(gz.zw)
	return err
}

type gzipCompressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func NewGzipCompressReader(r io.ReadCloser) (*gzipCompressReader, error) {
	zr, err := acquireGzipReader(r)
	if err != nil {
		return nil, err
	}

	return &gzipCompressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (gz *gzipCompressReader) Read(p []byte) (int, error) {
	return gz.zr.Read(p)
}
func (gz *gzipCompressReader) Close() error {
	err := gz.r.Close()
	releaseGzipReader(gz.zr)
	return err
}
