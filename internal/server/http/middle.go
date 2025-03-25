package http_internal

import (
	"go.uber.org/zap"
	"net/http"
	"time"
)

func loggingMiddleware(next http.HandlerFunc, log Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		next.ServeHTTP(w, r)
		log.GetZapLogger().With(
			zap.String("Client IP", r.RemoteAddr),
			zap.String("Request Date", time.Now().String()),
			zap.String("Request Method", r.Method),
			zap.String("Request URI", r.RequestURI),
			zap.String("Request UserAgent", r.UserAgent()),
			zap.String("Request Scheme", r.URL.Scheme),
			zap.String("Request Status", w.Header().Get("X-Request-Status")),
			zap.String("Request in work for", time.Since(t).String()),
		).Info("http middle log")
		errHeader := w.Header().Get("X-Request-Error")
		if errHeader != "" {
			log.Error("Error at middle logging: " + errHeader)
		}
	}
}
