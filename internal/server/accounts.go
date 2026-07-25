package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ctrl-research/getbud/internal/auth"
	"github.com/ctrl-research/getbud/internal/store"
	"github.com/ctrl-research/getbud/internal/store/sqlcgen"
)

var accountTypes = map[string]store.AccountType{
	"chequing":       sqlcgen.AccountTypeChequing,
	"savings":        sqlcgen.AccountTypeSavings,
	"credit_card":    sqlcgen.AccountTypeCreditCard,
	"rrsp":           sqlcgen.AccountTypeRrsp,
	"tfsa":           sqlcgen.AccountTypeTfsa,
	"fhsa":           sqlcgen.AccountTypeFhsa,
	"non_registered": sqlcgen.AccountTypeNonRegistered,
	"other":          sqlcgen.AccountTypeOther,
}

type accountJSON struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	Type                string    `json:"type"`
	Currency            string    `json:"currency"`
	Institution         string    `json:"institution"`
	OpeningBalanceCents int64     `json:"openingBalanceCents"`
	IsArchived          bool      `json:"isArchived"`
	// IsInvestment marks snapshot-tracked account types (balance is the
	// latest snapshot rather than a transaction sum).
	IsInvestment bool      `json:"isInvestment"`
	BalanceCents int64     `json:"balanceCents"`
	CreatedAt    time.Time `json:"createdAt"`
}

func toAccountJSON(a store.Account, balanceCents int64) accountJSON {
	return accountJSON{
		ID: a.ID, Name: a.Name, Type: string(a.Type),
		Currency: strings.TrimSpace(a.Currency), Institution: a.Institution,
		OpeningBalanceCents: a.OpeningBalanceCents, IsArchived: a.IsArchived,
		IsInvestment: store.InvestmentTypes[a.Type],
		BalanceCents: balanceCents, CreatedAt: a.CreatedAt,
	}
}

func (api *budgetAPI) listAccounts(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	accounts, err := api.accounts.List(r.Context(), user.ID)
	if err != nil {
		apiInternalError(w, "list accounts", err)
		return
	}
	out := make([]accountJSON, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, toAccountJSON(a.Account, a.BalanceCents))
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": out})
}

func (api *budgetAPI) createAccount(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var req struct {
		Name                string `json:"name"`
		Type                string `json:"type"`
		Currency            string `json:"currency"`
		Institution         string `json:"institution"`
		OpeningBalanceCents int64  `json:"openingBalanceCents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		apiError(w, http.StatusBadRequest, "bad_request", "name and type are required")
		return
	}
	typ, ok := accountTypes[req.Type]
	if !ok {
		apiError(w, http.StatusBadRequest, "invalid_type", "unknown account type")
		return
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "CAD"
	}
	if len(currency) != 3 {
		apiError(w, http.StatusBadRequest, "invalid_currency", "currency must be a 3-letter code")
		return
	}
	account, err := api.accounts.Create(r.Context(), user.ID, strings.TrimSpace(req.Name), typ, currency, strings.TrimSpace(req.Institution), req.OpeningBalanceCents)
	if errors.Is(err, store.ErrConflict) {
		apiError(w, http.StatusConflict, "duplicate_name", "an account with that name already exists")
		return
	}
	if err != nil {
		apiInternalError(w, "create account", err)
		return
	}
	writeJSON(w, http.StatusCreated, toAccountJSON(account, account.OpeningBalanceCents))
}

func (api *budgetAPI) updateAccount(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	id, ok := pathUUID(r, "id")
	if !ok {
		apiError(w, http.StatusNotFound, "not_found", "account not found")
		return
	}
	account, err := api.accounts.ByID(r.Context(), user.ID, id)
	if errors.Is(err, store.ErrNotFound) {
		apiError(w, http.StatusNotFound, "not_found", "account not found")
		return
	}
	if err != nil {
		apiInternalError(w, "load account", err)
		return
	}

	var req struct {
		Name                *string `json:"name"`
		Type                *string `json:"type"`
		Currency            *string `json:"currency"`
		Institution         *string `json:"institution"`
		OpeningBalanceCents *int64  `json:"openingBalanceCents"`
		IsArchived          *bool   `json:"isArchived"`
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
		account.Name = strings.TrimSpace(*req.Name)
	}
	if req.Type != nil {
		typ, ok := accountTypes[*req.Type]
		if !ok {
			apiError(w, http.StatusBadRequest, "invalid_type", "unknown account type")
			return
		}
		account.Type = typ
	}
	if req.Currency != nil {
		currency := strings.ToUpper(strings.TrimSpace(*req.Currency))
		if len(currency) != 3 {
			apiError(w, http.StatusBadRequest, "invalid_currency", "currency must be a 3-letter code")
			return
		}
		account.Currency = currency
	}
	if req.Institution != nil {
		account.Institution = strings.TrimSpace(*req.Institution)
	}
	if req.OpeningBalanceCents != nil {
		account.OpeningBalanceCents = *req.OpeningBalanceCents
	}
	if req.IsArchived != nil {
		account.IsArchived = *req.IsArchived
	}

	updated, err := api.accounts.Update(r.Context(), user.ID, account)
	if errors.Is(err, store.ErrConflict) {
		apiError(w, http.StatusConflict, "duplicate_name", "an account with that name already exists")
		return
	}
	if err != nil {
		apiInternalError(w, "update account", err)
		return
	}
	writeJSON(w, http.StatusOK, toAccountJSON(updated, 0))
}

func (api *budgetAPI) deleteAccount(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	id, ok := pathUUID(r, "id")
	if !ok {
		apiError(w, http.StatusNotFound, "not_found", "account not found")
		return
	}
	n, err := api.accounts.TransactionCount(r.Context(), user.ID, id)
	if err != nil {
		apiInternalError(w, "count account transactions", err)
		return
	}
	if n > 0 {
		apiError(w, http.StatusConflict, "account_not_empty", "account has transactions; archive it instead")
		return
	}
	switch err := api.accounts.Delete(r.Context(), user.ID, id); {
	case errors.Is(err, store.ErrNotFound):
		apiError(w, http.StatusNotFound, "not_found", "account not found")
	case err != nil:
		apiInternalError(w, "delete account", err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- balance snapshots -------------------------------------------------------

type snapshotJSON struct {
	ID           uuid.UUID `json:"id"`
	AccountID    uuid.UUID `json:"accountId"`
	AsOf         string    `json:"asOf"`
	BalanceCents int64     `json:"balanceCents"`
}

func toSnapshotJSON(s store.BalanceSnapshot) snapshotJSON {
	return snapshotJSON{ID: s.ID, AccountID: s.AccountID, AsOf: s.AsOf.Format(dateOnly), BalanceCents: s.BalanceCents}
}

// ownAccount loads an account owned by the user, writing the 404 itself when
// missing.
func (api *budgetAPI) ownAccount(w http.ResponseWriter, r *http.Request, userID, accountID uuid.UUID) (store.Account, bool) {
	account, err := api.accounts.ByID(r.Context(), userID, accountID)
	if errors.Is(err, store.ErrNotFound) {
		apiError(w, http.StatusNotFound, "not_found", "account not found")
		return store.Account{}, false
	}
	if err != nil {
		apiInternalError(w, "load account", err)
		return store.Account{}, false
	}
	return account, true
}

func (api *budgetAPI) listSnapshots(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	accountID, ok := pathUUID(r, "id")
	if !ok {
		apiError(w, http.StatusNotFound, "not_found", "account not found")
		return
	}
	if _, ok := api.ownAccount(w, r, user.ID, accountID); !ok {
		return
	}
	snaps, err := api.snapshots.List(r.Context(), user.ID, accountID)
	if err != nil {
		apiInternalError(w, "list snapshots", err)
		return
	}
	out := make([]snapshotJSON, 0, len(snaps))
	for _, s := range snaps {
		out = append(out, toSnapshotJSON(s))
	}
	writeJSON(w, http.StatusOK, map[string]any{"snapshots": out})
}

func (api *budgetAPI) upsertSnapshot(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	accountID, ok := pathUUID(r, "id")
	if !ok {
		apiError(w, http.StatusNotFound, "not_found", "account not found")
		return
	}
	if _, ok := api.ownAccount(w, r, user.ID, accountID); !ok {
		return
	}
	var req struct {
		AsOf         string `json:"asOf"`
		BalanceCents int64  `json:"balanceCents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
		return
	}
	asOf, ok := parseDate(req.AsOf)
	if !ok {
		apiError(w, http.StatusBadRequest, "invalid_date", "asOf must be YYYY-MM-DD")
		return
	}
	snap, err := api.snapshots.Upsert(r.Context(), user.ID, accountID, asOf, req.BalanceCents)
	if err != nil {
		apiInternalError(w, "upsert snapshot", err)
		return
	}
	writeJSON(w, http.StatusOK, toSnapshotJSON(snap))
}

func (api *budgetAPI) deleteSnapshot(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	id, ok := pathUUID(r, "snapshotId")
	if !ok {
		apiError(w, http.StatusNotFound, "not_found", "snapshot not found")
		return
	}
	switch err := api.snapshots.Delete(r.Context(), user.ID, id); {
	case errors.Is(err, store.ErrNotFound):
		apiError(w, http.StatusNotFound, "not_found", "snapshot not found")
	case err != nil:
		apiInternalError(w, "delete snapshot", err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
