package main

import (
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/grimdork/climate/fx"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

func statusColour(code int) string {
	switch {
	case code >= 500:
		return fx.Sprint("{danger}{}{@}", code)
	case code >= 400:
		return fx.Sprint("{warning}{}{@}", code)
	case code >= 300:
		return fx.Sprint("{info}{}{@}", code)
	default:
		return fx.Sprint("{success}{}{@}", code)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)
		fx.Println("{logstamp} {} {} {} {}", r.Method, r.URL.Path, statusColour(rw.status), time.Since(start))
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				fx.Fprintln(os.Stderr, "{logstamp} {danger}panic:{@} {}\n{}", err, debug.Stack())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
