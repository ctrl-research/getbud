package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ctrl-research/getbud/internal/auth"
	"github.com/ctrl-research/getbud/internal/store"
	"github.com/ctrl-research/getbud/internal/store/sqlcgen"
)

type transactionJSON struct {
	ID            uuid.UUID  `json:"id"`
	AccountID     uuid.UUID  `json:"accountId"`
	AccountName   string     `json:"accountName,omitempty"`
	Date          string     `json:"date"`
	AmountCents   int64      `json:"amountCents"`
	Payee         string     `json:"payee"`
	Notes         string     `json:"notes"`
	CategoryID    *uuid.UUID `json:"categoryId"`
	CategoryName  string     `json:"categoryName,omitempty"`
	CategoryKind  string     `json:"categoryKind,omitempty"`
	CategoryColor string     `json:"categoryColor,omitempty"`
	IsTransfer    bool       `json:"isTransfer"`
	ImportBatchID *uuid.UUID `json:"importBatchId,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}

func toTransactionJSON(t store.Transaction) transactionJSON {
	return transactionJSON{
		ID: t.ID, AccountID: t.AccountID, Date: t.Date.Format(dateOnly),
		AmountCents: t.AmountCents, Payee: t.Payee, Notes: t.Notes,
		CategoryID: t.CategoryID, IsTransfer: t.TransferGroupID != nil,
		ImportBatchID: t.ImportBatchID, CreatedAt: t.CreatedAt,
	}
}

func toTransactionRowJSON(t store.TransactionRow) transactionJSON {
	out := transactionJSON{
		ID: t.ID, AccountID: t.AccountID, AccountName: t.AccountName,
		Date: t.Date.Format(dateOnly), AmountCents: t.AmountCents,
		Payee: t.Payee, Notes: t.Notes, CategoryID: t.CategoryID,
		IsTransfer: t.TransferGroupID != nil, ImportBatchID: t.ImportBatchID,
		CreatedAt: t.CreatedAt,
	}
	if t.CategoryName != nil {
		out.CategoryName = *t.CategoryName
	}
	if t.CategoryKind != nil {
		out.CategoryKind = string(*t.CategoryKind)
	}
	if t.CategoryColor != nil {
		out.CategoryColor = *t.CategoryColor
	}
	return out
}

// listTransactions handles GET /api/v1/transactions with optional filters:
// from, to (dates), accountId, categoryId, uncategorized, q, limit, offset.
func (api *budgetAPI) listTransactions(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	q := r.URL.Query()

	params := store.ListParams{UserID: user.ID, RowLimit: 50}
	if v := q.Get("from"); v != "" {
		d, ok := parseDate(v)
		if !ok {
			apiError(w, http.StatusBadRequest, "invalid_date", "from must be YYYY-MM-DD")
			return
		}
		params.FromDate = &d
	}
	if v := q.Get("to"); v != "" {
		d, ok := parseDate(v)
		if !ok {
			apiError(w, http.StatusBadRequest, "invalid_date", "to must be YYYY-MM-DD")
			return
		}
		params.ToDate = &d
	}
	if v := q.Get("accountId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			apiError(w, http.StatusBadRequest, "bad_request", "accountId must be a uuid")
			return
		}
		params.AccountID = &id
	}
	if v := q.Get("categoryId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			apiError(w, http.StatusBadRequest, "bad_request", "categoryId must be a uuid")
			return
		}
		params.CategoryID = &id
	}
	params.Uncategorized = q.Get("uncategorized") == "true"
	params.Search = strings.TrimSpace(q.Get("q"))
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			apiError(w, http.StatusBadRequest, "bad_request", "limit must be 1-500")
			return
		}
		params.RowLimit = int32(n)
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			apiError(w, http.StatusBadRequest, "bad_request", "offset must be >= 0")
			return
		}
		params.RowOffset = int32(n)
	}

	rows, err := api.transactions.List(r.Context(), params)
	if err != nil {
		apiInternalError(w, "list transactions", err)
		return
	}
	total, err := api.transactions.Count(r.Context(), params)
	if err != nil {
		apiInternalError(w, "count transactions", err)
		return
	}
	out := make([]transactionJSON, 0, len(rows))
	for _, t := range rows {
		out = append(out, toTransactionRowJSON(t))
	}
	writeJSON(w, http.StatusOK, map[string]any{"transactions": out, "totalCount": total})
}

// ownCategory validates that a category belongs to the user, writing the
// error response itself on failure.
func (api *budgetAPI) ownCategory(w http.ResponseWriter, r *http.Request, userID, categoryID uuid.UUID) bool {
	if _, err := api.categories.ByID(r.Context(), userID, categoryID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			apiError(w, http.StatusBadRequest, "bad_request", "category not found")
		} else {
			apiInternalError(w, "load category", err)
		}
		return false
	}
	return true
}

func (api *budgetAPI) createTransaction(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var req struct {
		AccountID   uuid.UUID  `json:"accountId"`
		Date        string     `json:"date"`
		AmountCents int64      `json:"amountCents"`
		Payee       string     `json:"payee"`
		Notes       string     `json:"notes"`
		CategoryID  *uuid.UUID `json:"categoryId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
		return
	}
	date, ok := parseDate(req.Date)
	if !ok {
		apiError(w, http.StatusBadRequest, "invalid_date", "date must be YYYY-MM-DD")
		return
	}
	if req.AmountCents == 0 {
		apiError(w, http.StatusBadRequest, "invalid_amount", "amount cannot be zero")
		return
	}
	if _, ok := api.ownAccount(w, r, user.ID, req.AccountID); !ok {
		return
	}
	if req.CategoryID != nil && !api.ownCategory(w, r, user.ID, *req.CategoryID) {
		return
	}
	t, err := api.transactions.Create(r.Context(), sqlcgen.CreateTransactionParams{
		UserID: user.ID, AccountID: req.AccountID, Date: date,
		AmountCents: req.AmountCents, Payee: strings.TrimSpace(req.Payee),
		Notes: strings.TrimSpace(req.Notes), CategoryID: req.CategoryID,
	})
	if err != nil {
		apiInternalError(w, "create transaction", err)
		return
	}
	writeJSON(w, http.StatusCreated, toTransactionJSON(t))
}

func (api *budgetAPI) updateTransaction(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	id, ok := pathUUID(r, "id")
	if !ok {
		apiError(w, http.StatusNotFound, "not_found", "transaction not found")
		return
	}
	t, err := api.transactions.ByID(r.Context(), user.ID, id)
	if errors.Is(err, store.ErrNotFound) {
		apiError(w, http.StatusNotFound, "not_found", "transaction not found")
		return
	}
	if err != nil {
		apiInternalError(w, "load transaction", err)
		return
	}
	if t.TransferGroupID != nil {
		apiError(w, http.StatusConflict, "transfer_readonly", "transfer legs cannot be edited; delete the transfer and recreate it")
		return
	}

	var req struct {
		Date        *string    `json:"date"`
		AmountCents *int64     `json:"amountCents"`
		Payee       *string    `json:"payee"`
		Notes       *string    `json:"notes"`
		CategoryID  *uuid.UUID `json:"categoryId"`
		// ClearCategory distinguishes "set uncategorized" from "leave as is"
		// (both encode categoryId as null in JSON).
		ClearCategory bool `json:"clearCategory"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
		return
	}
	if req.Date != nil {
		d, ok := parseDate(*req.Date)
		if !ok {
			apiError(w, http.StatusBadRequest, "invalid_date", "date must be YYYY-MM-DD")
			return
		}
		t.Date = d
	}
	if req.AmountCents != nil {
		if *req.AmountCents == 0 {
			apiError(w, http.StatusBadRequest, "invalid_amount", "amount cannot be zero")
			return
		}
		t.AmountCents = *req.AmountCents
	}
	if req.Payee != nil {
		t.Payee = strings.TrimSpace(*req.Payee)
	}
	if req.Notes != nil {
		t.Notes = strings.TrimSpace(*req.Notes)
	}
	if req.ClearCategory {
		t.CategoryID = nil
	} else if req.CategoryID != nil {
		if !api.ownCategory(w, r, user.ID, *req.CategoryID) {
			return
		}
		t.CategoryID = req.CategoryID
	}

	updated, err := api.transactions.Update(r.Context(), user.ID, t)
	if err != nil {
		apiInternalError(w, "update transaction", err)
		return
	}
	writeJSON(w, http.StatusOK, toTransactionJSON(updated))
}

func (api *budgetAPI) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	id, ok := pathUUID(r, "id")
	if !ok {
		apiError(w, http.StatusNotFound, "not_found", "transaction not found")
		return
	}
	switch err := api.transactions.Delete(r.Context(), user.ID, id); {
	case errors.Is(err, store.ErrNotFound):
		apiError(w, http.StatusNotFound, "not_found", "transaction not found")
	case err != nil:
		apiInternalError(w, "delete transaction", err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// createTransfer inserts both legs of a transfer between two of the user's
// own accounts; transfers never count as income or expense.
func (api *budgetAPI) createTransfer(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var req struct {
		FromAccountID uuid.UUID `json:"fromAccountId"`
		ToAccountID   uuid.UUID `json:"toAccountId"`
		Date          string    `json:"date"`
		AmountCents   int64     `json:"amountCents"`
		Notes         string    `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
		return
	}
	date, ok := parseDate(req.Date)
	if !ok {
		apiError(w, http.StatusBadRequest, "invalid_date", "date must be YYYY-MM-DD")
		return
	}
	if req.AmountCents <= 0 {
		apiError(w, http.StatusBadRequest, "invalid_amount", "amount must be positive")
		return
	}
	if req.FromAccountID == req.ToAccountID {
		apiError(w, http.StatusBadRequest, "bad_request", "cannot transfer within the same account")
		return
	}
	if _, ok := api.ownAccount(w, r, user.ID, req.FromAccountID); !ok {
		return
	}
	if _, ok := api.ownAccount(w, r, user.ID, req.ToAccountID); !ok {
		return
	}
	legs, err := api.transactions.CreateTransfer(r.Context(), user.ID, req.FromAccountID, req.ToAccountID, date, req.AmountCents, strings.TrimSpace(req.Notes))
	if err != nil {
		apiInternalError(w, "create transfer", err)
		return
	}
	out := make([]transactionJSON, 0, len(legs))
	for _, t := range legs {
		out = append(out, toTransactionJSON(t))
	}
	writeJSON(w, http.StatusCreated, map[string]any{"transactions": out})
}
