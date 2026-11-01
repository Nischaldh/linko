package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
)

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler{
	return func(next http.Handler) http.Handler{
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info(fmt.Sprintf("Served request: %v %v", r.Method, r.URL.Path))
			next.ServeHTTP(w, r)
		})
	}
}