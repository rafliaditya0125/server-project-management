package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/rafliaditya0125/server-project-management/pkg/response"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %v\n%s", err, debug.Stack())
				response.Error(w, http.StatusInternalServerError, "Internal server error", "panic occurred")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
