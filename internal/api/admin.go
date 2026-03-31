package api

import (
	"net/http"

	"github.com/papi-api/papi/internal/model"
)

func adminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sub, ok := r.Context().Value(subscriberKey).(*model.Subscriber)
		if !ok || !sub.IsAdmin {
			writeError(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
