package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/ctrl-research/getbud/internal/auth"
	"github.com/ctrl-research/getbud/internal/store"
	"github.com/ctrl-research/getbud/internal/store/sqlcgen"
)

// registeredTypes are the account types with CRA contribution limits.
var registeredTypes = map[string]store.AccountType{
	"rrsp": sqlcgen.AccountTypeRrsp,
	"tfsa": sqlcgen.AccountTypeTfsa,
	"fhsa": sqlcgen.AccountTypeFhsa,
}

// contributionHints are the published annual dollar limits, shown as
// placeholder hints only - actual room is personal (carry-forward,
// withdrawals, income-based RRSP room) and must be entered by the user.
// For years beyond the map the latest known year's value is used.
var contributionHints = map[string]map[int]int64{
	"tfsa": {2023: 6_500_00, 2024: 7_000_00, 2025: 7_000_00},
	"rrsp": {2023: 30_780_00, 2024: 31_560_00, 2025: 32_490_00},
	"fhsa": {2023: 8_000_00, 2024: 8_000_00, 2025: 8_000_00},
}

func hintFor(typ string, year int) int64 {
	years := contributionHints[typ]
	if v, ok := years[year]; ok {
		return v
	}
	// Fall back to the most recent published year.
	var latestYear int
	var latest int64
	for y, v := range years {
		if y > latestYear {
			latestYear, latest = y, v
		}
	}
	return latest
}

type contributionTypeJSON struct {
	RoomCents        *int64 `json:"roomCents"`
	Notes            string `json:"notes"`
	ContributedCents int64  `json:"contributedCents"`
	WithdrawnCents   int64  `json:"withdrawnCents"`
	RemainingCents   *int64 `json:"remainingCents"`
	DefaultHintCents int64  `json:"defaultHintCents"`
}

// getContributionRoom returns room, derived contributions/withdrawals, and
// remaining space per registered type for a tax year.
func (api *budgetAPI) getContributionRoom(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	year := time.Now().Year()
	if v := r.URL.Query().Get("year"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 2000 || n > 2100 {
			apiError(w, http.StatusBadRequest, "bad_request", "year must be a 4-digit year")
			return
		}
		year = n
	}

	rooms, err := api.contributions.ListRoom(r.Context(), user.ID, year)
	if err != nil {
		apiInternalError(w, "list contribution room", err)
		return
	}
	totals, err := api.contributions.Totals(r.Context(), user.ID, year)
	if err != nil {
		apiInternalError(w, "contribution totals", err)
		return
	}

	out := make(map[string]contributionTypeJSON, len(registeredTypes))
	for name := range registeredTypes {
		out[name] = contributionTypeJSON{DefaultHintCents: hintFor(name, year)}
	}
	for _, room := range rooms {
		entry := out[string(room.AccountType)]
		v := room.RoomCents
		entry.RoomCents = &v
		entry.Notes = room.Notes
		out[string(room.AccountType)] = entry
	}
	for _, t := range totals {
		entry := out[string(t.AccountType)]
		entry.ContributedCents = t.ContributedCents
		entry.WithdrawnCents = t.WithdrawnCents
		out[string(t.AccountType)] = entry
	}
	for name, entry := range out {
		if entry.RoomCents != nil {
			remaining := *entry.RoomCents - entry.ContributedCents
			entry.RemainingCents = &remaining
			out[name] = entry
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"year": year, "types": out})
}

// putContributionRoom upserts the user-entered room for a type and year.
func (api *budgetAPI) putContributionRoom(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	typ, ok := registeredTypes[r.PathValue("type")]
	if !ok {
		apiError(w, http.StatusNotFound, "not_found", "type must be rrsp, tfsa, or fhsa")
		return
	}
	year, err := strconv.Atoi(r.PathValue("year"))
	if err != nil || year < 2000 || year > 2100 {
		apiError(w, http.StatusBadRequest, "bad_request", "year must be a 4-digit year")
		return
	}
	var req struct {
		RoomCents int64  `json:"roomCents"`
		Notes     string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
		return
	}
	if req.RoomCents < 0 {
		apiError(w, http.StatusBadRequest, "invalid_amount", "room cannot be negative")
		return
	}
	room, err := api.contributions.UpsertRoom(r.Context(), user.ID, typ, year, req.RoomCents, req.Notes)
	if err != nil {
		apiInternalError(w, "upsert contribution room", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"accountType": string(room.AccountType),
		"taxYear":     room.TaxYear,
		"roomCents":   room.RoomCents,
		"notes":       room.Notes,
	})
}
