package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zapi-api/zapi/internal/store"
)

type SubscriberHandlers struct {
	store store.Store
}

func NewSubscriberHandlers(s store.Store) *SubscriberHandlers {
	return &SubscriberHandlers{store: s}
}

type createSubscriberRequest struct {
	Name       string `json:"name"`
	WebhookURL string `json:"webhook_url"`
}

func (h *SubscriberHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var req createSubscriberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	sub, err := h.store.CreateSubscriber(r.Context(), req.Name, req.WebhookURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create subscriber")
		return
	}
	writeJSON(w, http.StatusCreated, sub)
}

func (h *SubscriberHandlers) List(w http.ResponseWriter, r *http.Request) {
	subs, err := h.store.GetSubscribers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list subscribers")
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

type updateSubscriberRequest struct {
	Name       string `json:"name"`
	WebhookURL string `json:"webhook_url"`
}

func (h *SubscriberHandlers) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateSubscriberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := h.store.UpdateSubscriber(r.Context(), id, req.Name, req.WebhookURL); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update subscriber")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *SubscriberHandlers) Deactivate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeactivateSubscriber(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to deactivate subscriber")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}
