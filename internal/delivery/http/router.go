package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rafliaditya0125/server-project-management/internal/delivery/http/handler"
	"github.com/rafliaditya0125/server-project-management/internal/delivery/http/middleware"
	"github.com/rafliaditya0125/server-project-management/pkg/response"
)

func NewRouter(appHandler *handler.AppHandler, setupHandler *handler.SetupHandler) http.Handler {
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)
	r.Use(middleware.CORS())

	// Health Check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, "Server is healthy", map[string]string{"status": "ok"})
	})

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// App management
		r.Route("/apps", func(r chi.Router) {
			r.Get("/", appHandler.ListApps)
			r.Post("/", appHandler.CreateApp)
			r.Get("/{name}", appHandler.GetApp)
			r.Delete("/{name}", appHandler.DeleteApp)
			r.Get("/{name}/logs", appHandler.GetLogs)
			r.Post("/{name}/manage", appHandler.ManageService)
		})

		// Setup management
		r.Route("/setup", func(r chi.Router) {
			r.Get("/status", setupHandler.GetStatus)
			r.Post("/", setupHandler.ExecuteSetup)
			r.Post("/fastcgi", setupHandler.ConfigureFastCGI)
		})
	})

	return r
}
