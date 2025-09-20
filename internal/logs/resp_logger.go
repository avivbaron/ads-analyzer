package logs

import "net/http"

// respLogger captures status and bytes written
type RespLogger struct {
	http.ResponseWriter
	Status int
	Bytes  int
}

func (l *RespLogger) WriteHeader(code int) {
	l.Status = code
	l.ResponseWriter.WriteHeader(code)
}

// Write forwards the response body to the underlying writer and tracks
// how many bytes have been sent so Bytes mirrors the payload size.
func (l *RespLogger) Write(b []byte) (int, error) {
	n, err := l.ResponseWriter.Write(b)
	l.Bytes += n
	return n, err
}
