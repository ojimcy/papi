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

	auth := NewAuthMiddleware(s)
	h := NewHandlers(c, s)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/v1", func(r chi.Router) {
		r.Use(auth.Handler)
		r.Get("/rates", h.GetRates)
		r.Get("/rates/history", h.GetHistory)
	})

	return r
}
