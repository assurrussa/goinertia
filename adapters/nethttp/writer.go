package nethttp

import (
	"bufio"
	"io"
	"net"
	"net/http"

	"github.com/assurrussa/goinertia/core"
)

// appendVary preserves every field line, including wildcard and existing tokens.
func appendVary(h http.Header, token string) {
	values := h.Values("Vary")
	for _, value := range values {
		if core.HasVaryToken(value, token) {
			return
		}
	}
	if len(values) <= 1 {
		h.Set("Vary", core.VaryValue(h.Get("Vary"), token))
		return
	}
	h.Add("Vary", token)
}

type responseWriter struct {
	http.ResponseWriter
	before   func(int) int
	wrote    bool
	dropBody bool
}

func (w *responseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *responseWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}
	if status >= 100 && status < 200 && status != 101 {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	normalized := w.before(status)
	w.dropBody = normalized != status && normalized == http.StatusSeeOther
	if w.dropBody {
		w.Header().Del("Content-Length")
	}
	w.wrote = true
	w.ResponseWriter.WriteHeader(normalized)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	if w.dropBody {
		return len(body), nil
	}
	return w.ResponseWriter.Write(body)
}

func (w *responseWriter) WriteString(body string) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	if w.dropBody {
		return len(body), nil
	}
	if writer, ok := w.ResponseWriter.(io.StringWriter); ok {
		return writer.WriteString(body)
	}
	return w.ResponseWriter.Write([]byte(body))
}

func (w *responseWriter) ReadFrom(r io.Reader) (int64, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	if w.dropBody {
		return io.Copy(io.Discard, r)
	}
	if reader, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return reader.ReadFrom(r)
	}
	return io.Copy(struct{ io.Writer }{w.ResponseWriter}, r)
}

func (w *responseWriter) finish() {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
}

func (w *responseWriter) FlushError() error {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return http.NewResponseController(w.ResponseWriter).Flush()
}

type flushWriter struct{ *responseWriter }

func (w flushWriter) Flush() { _ = w.FlushError() }

type hijackWriter struct{ *responseWriter }

func (w hijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	// Hijacking does not commit an HTTP response. The caller owns its connection.
	conn, rw, err := http.NewResponseController(w.ResponseWriter).Hijack()
	if err == nil {
		w.wrote = true
	}
	return conn, rw, err
}

type pushWriter struct{ *responseWriter }

func (w pushWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func committed(w http.ResponseWriter) bool {
	for w != nil {
		if value, ok := w.(interface{ isCommitted() bool }); ok {
			return value.isCommitted()
		}
		unwrapper, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return false
		}
		w = unwrapper.Unwrap()
	}
	return false
}
func (w *responseWriter) isCommitted() bool { return w.wrote }

type (
	closeNotifier interface{ CloseNotify() <-chan bool }
	notifyWriter  struct{ *responseWriter }
)

func (w notifyWriter) CloseNotify() <-chan bool {
	n, ok := w.ResponseWriter.(closeNotifier)
	if !ok {
		return nil
	}
	return n.CloseNotify()
}

// Optional writer interfaces are advertised only when the underlying writer
// has them. ResponseController follows Unwrap for deadlines and full duplex.
func preserveCapabilities(w *responseWriter) http.ResponseWriter {
	mask := 0
	if _, ok := w.ResponseWriter.(http.Flusher); ok {
		mask |= 1
	}
	if _, ok := w.ResponseWriter.(http.Hijacker); ok {
		mask |= 2
	}
	if _, ok := w.ResponseWriter.(http.Pusher); ok {
		mask |= 4
	}
	if _, ok := w.ResponseWriter.(closeNotifier); ok {
		mask |= 8
	}
	switch mask {
	case 1:
		return struct {
			*responseWriter
			http.Flusher
		}{w, flushWriter{w}}
	case 2:
		return struct {
			*responseWriter
			http.Hijacker
		}{w, hijackWriter{w}}
	case 3:
		return struct {
			*responseWriter
			http.Flusher
			http.Hijacker
		}{w, flushWriter{w}, hijackWriter{w}}
	case 4:
		return struct {
			*responseWriter
			http.Pusher
		}{w, pushWriter{w}}
	case 5:
		return struct {
			*responseWriter
			http.Flusher
			http.Pusher
		}{w, flushWriter{w}, pushWriter{w}}
	case 6:
		return struct {
			*responseWriter
			http.Hijacker
			http.Pusher
		}{w, hijackWriter{w}, pushWriter{w}}
	case 7:
		return struct {
			*responseWriter
			http.Flusher
			http.Hijacker
			http.Pusher
		}{w, flushWriter{w}, hijackWriter{w}, pushWriter{w}}
	case 8:
		return struct {
			*responseWriter
			closeNotifier
		}{w, notifyWriter{w}}
	case 9:
		return struct {
			*responseWriter
			http.Flusher
			closeNotifier
		}{w, flushWriter{w}, notifyWriter{w}}
	case 10:
		return struct {
			*responseWriter
			http.Hijacker
			closeNotifier
		}{w, hijackWriter{w}, notifyWriter{w}}
	case 11:
		return struct {
			*responseWriter
			http.Flusher
			http.Hijacker
			closeNotifier
		}{w, flushWriter{w}, hijackWriter{w}, notifyWriter{w}}
	case 12:
		return struct {
			*responseWriter
			http.Pusher
			closeNotifier
		}{w, pushWriter{w}, notifyWriter{w}}
	case 13:
		return struct {
			*responseWriter
			http.Flusher
			http.Pusher
			closeNotifier
		}{w, flushWriter{w}, pushWriter{w}, notifyWriter{w}}
	case 14:
		return struct {
			*responseWriter
			http.Hijacker
			http.Pusher
			closeNotifier
		}{w, hijackWriter{w}, pushWriter{w}, notifyWriter{w}}
	case 15:
		return struct {
			*responseWriter
			http.Flusher
			http.Hijacker
			http.Pusher
			closeNotifier
		}{w, flushWriter{w}, hijackWriter{w}, pushWriter{w}, notifyWriter{w}}
	default:
		return w
	}
}
