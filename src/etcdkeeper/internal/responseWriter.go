package internal

import "net/http"

//CompletableResponseWriter extend http.ResponseWriter with completed state
// a ResponseWriter is Completed when method Write or WriteHeader have been called
type CompletableResponseWriter struct {
	http.ResponseWriter
	done bool
}

//New initialize CompletableResponseWriter with non completed http.ResponseWriter
func NewCompletableResponseWriter(w http.ResponseWriter) *CompletableResponseWriter {
	return &CompletableResponseWriter{w, false}
}

func (w *CompletableResponseWriter) WriteHeader(status int) {
	w.done = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *CompletableResponseWriter) Write(b []byte) (int, error) {
	w.done = true
	return w.ResponseWriter.Write(b)
}

//IsCompleted return true if response writer write or writeheader have been called
func (w *CompletableResponseWriter) IsCompleted() bool {
	return w.done
}
