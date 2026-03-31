package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/papi-api/papi/internal/cache"
	"github.com/papi-api/papi/internal/store"
)

func NewRouter(c *cache.Cache, s store.Store) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(requestIDToHeader)
	r.Use(corsMiddleware)

	auth := NewAuthMiddleware(s)
	h := NewHandlers(c, s)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		dbStatus := "ok"
		code := http.StatusOK

		if err := s.Ping(r.Context()); err != nil {
			dbStatus = "unavailable"
			status = "degraded"
			code = http.StatusServiceUnavailable
		}

		writeJSON(w, code, map[string]string{
			"status":   status,
			"database": dbStatus,
		})
	})

	rl := NewRateLimiter(10, 20) // 10 req/s, burst 20

	r.Route("/v1", func(r chi.Router) {
		r.Use(auth.Handler)
		r.Use(rl.Handler)
		r.Get("/pairs", h.GetPairs)
		r.Get("/rates", h.GetRates)
		r.Get("/rates/history", h.GetHistory)

		sh := NewSubscriberHandlers(s)
		r.Route("/subscribers", func(r chi.Router) {
			r.Use(adminOnly)
			r.Get("/", sh.List)
			r.Post("/", sh.Create)
			r.Put("/{id}", sh.Update)
			r.Delete("/{id}", sh.Deactivate)
		})
	})

	return r
}

func requestIDToHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reqID := middleware.GetReqID(r.Context()); reqID != "" {
			w.Header().Set("X-Request-Id", reqID)
		}
		next.ServeHTTP(w, r)
	})
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
