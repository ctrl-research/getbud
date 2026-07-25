package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/ctrl-research/getbud/internal/auth"
	"github.com/ctrl-research/getbud/internal/store"
)

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type categoryJSON struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Kind       string    `json:"kind"`
	Color      string    `json:"color"`
	IsArchived bool      `json:"isArchived"`
}

func toCategoryJSON(c store.Category) categoryJSON {
	return categoryJSON{ID: c.ID, Name: c.Name, Kind: string(c.Kind), Color: c.Color, IsArchived: c.IsArchived}
}

func (api *budgetAPI) listCategories(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	cats, err := api.categories.List(r.Context(), user.ID)
	if err != nil {
		apiInternalError(w, "list categories", err)
		return
	}
	out := make([]categoryJSON, 0, len(cats))
	for _, c := range cats {
		out = append(out, toCategoryJSON(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": out})
}

func (api *budgetAPI) createCategory(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var req struct {
		Name  string `json:"name"`
		Kind  string `json:"kind"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		apiError(w, http.StatusBadRequest, "bad_request", "name and kind are required")
		return
	}
	var kind store.CategoryKind
	switch req.Kind {
	case "income":
		kind = store.KindIncome
	case "expense":
		kind = store.KindExpense
	default:
		apiError(w, http.StatusBadRequest, "invalid_kind", `kind must be "income" or "expense"`)
		return
	}
	if req.Color != "" && !hexColorRe.MatchString(req.Color) {
		apiError(w, http.StatusBadRequest, "invalid_color", "color must be #rrggbb")
		return
	}
	cat, err := api.categories.Create(r.Context(), user.ID, strings.TrimSpace(req.Name), kind, strings.ToLower(req.Color))
	if errors.Is(err, store.ErrConflict) {
		apiError(w, http.StatusConflict, "duplicate_name", "a category with that name already exists")
		return
	}
	if err != nil {
		apiInternalError(w, "create category", err)
		return
	}
	writeJSON(w, http.StatusCreated, toCategoryJSON(cat))
}

func (api *budgetAPI) updateCategory(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	id, ok := pathUUID(r, "id")
	if !ok {
		apiError(w, http.StatusNotFound, "not_found", "category not found")
		return
	}
	cat, err := api.categories.ByID(r.Context(), user.ID, id)
	if errors.Is(err, store.ErrNotFound) {
		apiError(w, http.StatusNotFound, "not_found", "category not found")
		return
	}
	if err != nil {
		apiInternalError(w, "load category", err)
		return
	}

	var req struct {
		Name       *string `json:"name"`
		Color      *string `json:"color"`
		IsArchived *bool   `json:"isArchived"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
		return
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			apiError(w, http.StatusBadRequest, "bad_request", "name cannot be empty")
			return
		}
		cat.Name = strings.TrimSpace(*req.Name)
	}
	if req.Color != nil {
		if *req.Color != "" && !hexColorRe.MatchString(*req.Color) {
			apiError(w, http.StatusBadRequest, "invalid_color", "color must be #rrggbb")
			return
		}
		cat.Color = strings.ToLower(*req.Color)
	}
	if req.IsArchived != nil {
		cat.IsArchived = *req.IsArchived
	}

	updated, err := api.categories.Update(r.Context(), user.ID, cat)
	if errors.Is(err, store.ErrConflict) {
		apiError(w, http.StatusConflict, "duplicate_name", "a category with that name already exists")
		return
	}
	if err != nil {
		apiInternalError(w, "update category", err)
		return
	}
	writeJSON(w, http.StatusOK, toCategoryJSON(updated))
}

// deleteCategory removes a category. With ?reassignTo=<id>, its transactions
// move to another category first; otherwise they go uncategorized.
func (api *budgetAPI) deleteCategory(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	id, ok := pathUUID(r, "id")
	if !ok {
		apiError(w, http.StatusNotFound, "not_found", "category not found")
		return
	}
	if target := r.URL.Query().Get("reassignTo"); target != "" {
		targetID, err := uuid.Parse(target)
		if err != nil {
			apiError(w, http.StatusBadRequest, "bad_request", "reassignTo must be a category id")
			return
		}
		if _, err := api.categories.ByID(r.Context(), user.ID, targetID); err != nil {
			apiError(w, http.StatusBadRequest, "bad_request", "reassignTo category not found")
			return
		}
		if _, err := api.categories.Reassign(r.Context(), user.ID, id, targetID); err != nil {
			apiInternalError(w, "reassign category", err)
			return
		}
	}
	switch err := api.categories.Delete(r.Context(), user.ID, id); {
	case errors.Is(err, store.ErrNotFound):
		apiError(w, http.StatusNotFound, "not_found", "category not found")
	case err != nil:
		apiInternalError(w, "delete category", err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
