package api

import (
	handlers "function/internal/api/handler"
	"function/pkg"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors" // Import the CORS package
)

func NewRouter(params *pkg.Params) *chi.Mux {
	handler := handlers.NewHandler(params)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"X-API-KEY", "Content-Type", "Authorization"},
		ExposedHeaders:   []string{},
		AllowCredentials: true,
	}))

	r.Group(func(r chi.Router) {
		r.HandleFunc("/health/{endpoint:readiness|liveness}", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(handler.AuthMiddleware)

		r.Route("/v1/template", func(r chi.Router) {
			r.Post("/url", handler.HandlerFunctionUrl)
		})
	})

	return r
}
