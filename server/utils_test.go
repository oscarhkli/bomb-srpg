package server

import (
	"fmt"
	"net/http"
)

// BrokenResponseWriter simulates a severed connection or dropped TCP socket mid-transit.
type BrokenResponseWriter struct {
	header http.Header
	Code   int
}

func (b *BrokenResponseWriter) Header() http.Header {
	if b.header == nil {
		b.header = make(http.Header)
	}
	return b.header
}

func (b *BrokenResponseWriter) Write(bytes []byte) (int, error) {
	return 0, fmt.Errorf("network connection systematically aborted by peer")
}

func (b *BrokenResponseWriter) WriteHeader(statusCode int) {
	if b.Code != 0 {
		return
	}
	b.Code = statusCode
}
