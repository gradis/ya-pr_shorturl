package middleware

import (
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var allowedGzipContentTypes = map[string]struct{}{
	"application/json": {},
	"text/html":        {},
}

type gzipResponseWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func Gzip() gin.HandlerFunc {
	return func(c *gin.Context) {
		if hasGzipEncoding(c.GetHeader("Content-Encoding")) {
			gzReader, err := gzip.NewReader(c.Request.Body)
			if err != nil {
				c.String(http.StatusBadRequest, "bad gzip body")
				c.Abort()
				return
			}

			defer gzReader.Close()
			c.Request.Body = gzReader
		}

		if acceptsGzip(c.GetHeader("Accept-Encoding")) {
			gzWriter := &gzipResponseWriter{
				ResponseWriter: c.Writer,
			}

			c.Writer = gzWriter

			defer func() {
				if gzWriter.writer != nil {
					_ = gzWriter.writer.Close()
				}
			}()
		}

		c.Next()
	}
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	if w.shouldCompress(data) {
		w.enableGzip()
		return w.writer.Write(data)
	}

	return w.ResponseWriter.Write(data)
}

func (w *gzipResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *gzipResponseWriter) shouldCompress(data []byte) bool {
	if w.writer != nil {
		return true
	}

	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
		w.Header().Set("Content-Type", contentType)
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	_, ok := allowedGzipContentTypes[mediaType]
	return ok
}

func (w *gzipResponseWriter) enableGzip() {
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Add("Vary", "Accept-Encoding")
	w.Header().Del("Content-Length")

	w.writer = gzip.NewWriter(w.ResponseWriter)
}

func hasGzipEncoding(header string) bool {
	for _, part := range strings.Split(header, ",") {
		if strings.TrimSpace(strings.ToLower(part)) == "gzip" {
			return true
		}
	}

	return false
}

func acceptsGzip(header string) bool {
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(strings.ToLower(part))

		if part == "gzip" {
			return true
		}

		if strings.HasPrefix(part, "gzip;") && !strings.Contains(part, "q=0") {
			return true
		}
	}

	return false
}

var _ io.Writer = (*gzipResponseWriter)(nil)
