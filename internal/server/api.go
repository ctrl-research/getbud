package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ctrl-research/getbud/internal/store"
)

// budgetAPI bundles the domain stores behind the /api/v1 handlers.
type budgetAPI struct {
	accounts      *store.Accounts
	categories    *store.Categories
	transactions  *store.Transactions
	snapshots     *store.Snapshots
	contributions *store.Contributions
	imports       *store.Imports
	reports       *store.Reports
}

func apiError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

func apiInternalError(w http.ResponseWriter, what string, err error) {
	slog.Error(what, "err", err)
	apiError(w, http.StatusInternalServerError, "internal", "something went wrong")
}

// dateOnly is the wire format for all dates (no times anywhere in the API).
const dateOnly = "2006-01-02"

func parseDate(s string) (time.Time, bool) {
	t, err := time.Parse(dateOnly, s)
	return t, err == nil
}

// pathUUID parses a path parameter as a UUID; a malformed id is treated the
// same as a missing resource.
func pathUUID(r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	return id, err == nil
}
