package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/junnotantra/backend-test/internal/model"
	"github.com/junnotantra/backend-test/internal/service"
)

type ItemService interface {
	CreateItem(context.Context, string, string, int64) (model.Item, error)
	ListItems(context.Context) ([]model.Item, error)
	GetItem(context.Context, int64) (model.Item, error)
	AdjustStock(context.Context, int64, int64) (model.Item, error)
}

type Handler struct{ service ItemService }

func NewHandler(s ItemService) http.Handler {
	h := &Handler{service: s}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /items", h.listItems)
	mux.HandleFunc("POST /items", h.createItem)
	mux.HandleFunc("GET /items/{id}", h.getItem)
	mux.HandleFunc("PATCH /items/{id}/stock", h.adjustStock)
	return mux
}

type itemRequest struct {
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

type stockRequest struct {
	Delta int64 `json:"delta"`
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) createItem(w http.ResponseWriter, r *http.Request) {
	var req itemRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	created, err := h.service.CreateItem(r.Context(), req.SKU, req.Name, req.Quantity)
	if errors.Is(err, service.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "sku and name are required; quantity must be non-negative")
		return
	}
	if errors.Is(err, service.ErrSKUAlreadyExists) {
		writeError(w, http.StatusBadRequest, "sku already exists")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) listItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListItems(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) getItem(w http.ResponseWriter, r *http.Request) {
	id, ok := itemID(w, r)
	if !ok {
		return
	}
	current, err := h.service.GetItem(r.Context(), id)
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, current)
}

func (h *Handler) adjustStock(w http.ResponseWriter, r *http.Request) {
	id, ok := itemID(w, r)
	if !ok {
		return
	}
	var req stockRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	current, err := h.service.AdjustStock(r.Context(), id, req.Delta)
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		if errors.Is(err, model.ErrNegativeStock) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, current)
}

func itemID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return 0, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
