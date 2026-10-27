package middleware

import (
	"log"
	"net/http"
)

func RequestLogger(logger *log.Logger) func(http.Handler) http.Handler{
	return func(next http.Handler) http.Handler{
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Printf("Served request: %v %v", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}