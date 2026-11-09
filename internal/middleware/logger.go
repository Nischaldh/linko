package middleware

import (
	"log/slog"
	"net/http"
)

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler{
	return func(next http.Handler) http.Handler{
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info("Served request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("client_ip", r.RemoteAddr),
		)
			next.ServeHTTP(w, r)
		})
	}
}