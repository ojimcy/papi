package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/zapi-api/zapi/internal/cache"
	"github.com/zapi-api/zapi/internal/store"
)

func NewRouter(c *cache.Cache, s store.Store) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(corsMiddleware)

	auth := NewAuthMiddleware(s)
	h := NewHandlers(c, s)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/v1", func(r chi.Router) {
		r.Use(auth.Handler)
		r.Get("/rates", h.GetRates)
		r.Get("/rates/history", h.GetHistory)

		sh := NewSubscriberHandlers(s)
		r.Get("/subscribers", sh.List)
		r.Post("/subscribers", sh.Create)
		r.Put("/subscribers/{id}", sh.Update)
		r.Delete("/subscribers/{id}", sh.Deactivate)
	})

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
