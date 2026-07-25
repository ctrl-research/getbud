package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ctrl-research/getbud/internal/auth"
	"github.com/ctrl-research/getbud/internal/csvimport"
	"github.com/ctrl-research/getbud/internal/store"
	"github.com/ctrl-research/getbud/internal/store/sqlcgen"
)

const maxImportBytes = 10 << 20
const maxImportRows = 5000

type importRowJSON struct {
	Index       int    `json:"index"`
	Date        string `json:"date,omitempty"`
	AmountCents int64  `json:"amountCents"`
	Payee       string `json:"payee"`
	Notes       string `json:"notes,omitempty"`
	// Status: ok | duplicate | duplicate_in_file | error
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	// Matched describes the existing transaction a duplicate collides with.
	Matched *matchedTxnJSON `json:"matched,omitempty"`
}

type matchedTxnJSON struct {
	Date        string `json:"date"`
	AmountCents int64  `json:"amountCents"`
	Payee       string `json:"payee"`
}

// readImportUpload pulls the shared multipart fields (file + accountId) and
// verifies account ownership.
func (api *budgetAPI) readImportUpload(w http.ResponseWriter, r *http.Request) (fileBytes []byte, filename string, accountID uuid.UUID, ok bool) {
	if err := r.ParseMultipartForm(maxImportBytes); err != nil {
		apiError(w, http.StatusBadRequest, "bad_request", "expected multipart form upload")
		return nil, "", uuid.Nil, false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad_request", "file is required")
		return nil, "", uuid.Nil, false
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxImportBytes+1))
	if err != nil {
		apiInternalError(w, "read upload", err)
		return nil, "", uuid.Nil, false
	}
	if len(raw) > maxImportBytes {
		apiError(w, http.StatusBadRequest, "too_large", "file exceeds 10 MB")
		return nil, "", uuid.Nil, false
	}

	accountID, err = uuid.Parse(r.FormValue("accountId"))
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad_request", "accountId is required")
		return nil, "", uuid.Nil, false
	}
	user, _ := auth.UserFrom(r.Context())
	if _, ok := api.ownAccount(w, r, user.ID, accountID); !ok {
		return nil, "", uuid.Nil, false
	}
	return raw, header.Filename, accountID, true
}

// previewImport parses the upload, guesses (or applies) a column mapping,
// and flags duplicates - nothing is written.
func (api *budgetAPI) previewImport(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	raw, _, accountID, ok := api.readImportUpload(w, r)
	if !ok {
		return
	}

	parsed, err := csvimport.Parse(bytes.NewReader(raw))
	if err != nil {
		apiError(w, http.StatusBadRequest, "parse_failed", err.Error())
		return
	}
	if len(parsed.Records) > maxImportRows {
		apiError(w, http.StatusBadRequest, "too_many_rows", fmt.Sprintf("file has %d rows (max %d)", len(parsed.Records), maxImportRows))
		return
	}

	var mapping csvimport.Mapping
	if m := r.FormValue("mapping"); m != "" {
		if err := json.Unmarshal([]byte(m), &mapping); err != nil {
			apiError(w, http.StatusBadRequest, "bad_request", "mapping is not valid JSON")
			return
		}
	} else if prev, err := api.imports.LatestMapping(r.Context(), user.ID, accountID); err == nil && prev != nil {
		if json.Unmarshal(prev, &mapping) != nil || mapping.DateColumn < 0 {
			mapping = csvimport.GuessMapping(parsed)
		}
	} else {
		mapping = csvimport.GuessMapping(parsed)
	}

	rows, err := api.buildImportRows(r, user.ID, accountID, parsed, mapping)
	if err != nil {
		apiInternalError(w, "check duplicates", err)
		return
	}

	// Raw sample rows help the mapping UI show real values per column.
	sample := parsed.Records
	if len(sample) > 5 {
		sample = sample[:5]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"headers":    parsed.Headers,
		"sampleRows": sample,
		"mapping":    mapping,
		"fileSha256": fileSHA(raw),
		"rowCount":   len(parsed.Records),
		"rows":       rows,
	})
}

// buildImportRows normalizes records and marks each row's status, comparing
// against both existing transactions and earlier rows in the same file.
func (api *budgetAPI) buildImportRows(r *http.Request, userID, accountID uuid.UUID, parsed *csvimport.File, mapping csvimport.Mapping) ([]importRowJSON, error) {
	normalized := csvimport.Normalize(parsed, mapping)

	hashes := make([][]byte, 0, len(normalized))
	for _, row := range normalized {
		if row.Err == "" {
			hashes = append(hashes, store.DedupHash(row.Date, row.AmountCents, row.Payee))
		}
	}
	existing := map[string]store.DedupMatch{}
	if len(hashes) > 0 {
		matches, err := api.transactions.FindByDedupHashes(r.Context(), userID, accountID, hashes)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			existing[string(m.DedupHash)] = m
		}
	}

	seenInFile := map[string]bool{}
	out := make([]importRowJSON, 0, len(normalized))
	for _, row := range normalized {
		j := importRowJSON{Index: row.Index, AmountCents: row.AmountCents, Payee: row.Payee, Notes: row.Notes}
		if row.Err != "" {
			j.Status = "error"
			j.Error = row.Err
			out = append(out, j)
			continue
		}
		j.Date = row.Date.Format(dateOnly)
		key := string(store.DedupHash(row.Date, row.AmountCents, row.Payee))
		switch {
		case seenInFile[key]:
			j.Status = "duplicate_in_file"
		case existing[key].ID != uuid.Nil:
			m := existing[key]
			j.Status = "duplicate"
			j.Matched = &matchedTxnJSON{Date: m.Date.Format(dateOnly), AmountCents: m.AmountCents, Payee: m.Payee}
		default:
			j.Status = "ok"
		}
		seenInFile[key] = true
		out = append(out, j)
	}
	return out, nil
}

// commitImport re-parses the upload and writes the batch. The client sends
// back the fileSha256 from preview; a mismatch means the file changed
// mid-wizard and the mapping can't be trusted.
func (api *budgetAPI) commitImport(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	raw, filename, accountID, ok := api.readImportUpload(w, r)
	if !ok {
		return
	}
	if expected := r.FormValue("fileSha256"); expected != "" && expected != fileSHA(raw) {
		apiError(w, http.StatusConflict, "file_changed", "the file changed since preview; start the import again")
		return
	}

	var mapping csvimport.Mapping
	if err := json.Unmarshal([]byte(r.FormValue("mapping")), &mapping); err != nil {
		apiError(w, http.StatusBadRequest, "bad_request", "mapping is required")
		return
	}
	excluded := map[int]bool{}
	if v := r.FormValue("excludedRows"); v != "" {
		var indexes []int
		if err := json.Unmarshal([]byte(v), &indexes); err != nil {
			apiError(w, http.StatusBadRequest, "bad_request", "excludedRows must be an array of row indexes")
			return
		}
		for _, i := range indexes {
			excluded[i] = true
		}
	}

	parsed, err := csvimport.Parse(bytes.NewReader(raw))
	if err != nil {
		apiError(w, http.StatusBadRequest, "parse_failed", err.Error())
		return
	}
	if len(parsed.Records) > maxImportRows {
		apiError(w, http.StatusBadRequest, "too_many_rows", fmt.Sprintf("file has %d rows (max %d)", len(parsed.Records), maxImportRows))
		return
	}

	normalized := csvimport.Normalize(parsed, mapping)
	var rows []sqlcgen.CreateTransactionParams
	skipped := 0
	for _, row := range normalized {
		if row.Err != "" || excluded[row.Index] {
			skipped++
			continue
		}
		rows = append(rows, sqlcgen.CreateTransactionParams{
			Date: row.Date, AmountCents: row.AmountCents, Payee: row.Payee, Notes: row.Notes,
		})
	}
	if len(rows) == 0 {
		apiError(w, http.StatusBadRequest, "nothing_to_import", "all rows were skipped or excluded")
		return
	}

	mappingJSON, err := json.Marshal(mapping)
	if err != nil {
		apiInternalError(w, "marshal mapping", err)
		return
	}
	batch, err := api.imports.Commit(r.Context(), user.ID, accountID, filename, mappingJSON, len(parsed.Records), skipped, rows)
	if err != nil {
		apiInternalError(w, "commit import", err)
		return
	}
	writeJSON(w, http.StatusCreated, toImportBatchJSON(batch, ""))
}

type importBatchJSON struct {
	ID            uuid.UUID `json:"id"`
	AccountID     uuid.UUID `json:"accountId"`
	AccountName   string    `json:"accountName,omitempty"`
	Filename      string    `json:"filename"`
	RowCount      int32     `json:"rowCount"`
	ImportedCount int32     `json:"importedCount"`
	SkippedCount  int32     `json:"skippedCount"`
	CreatedAt     time.Time `json:"createdAt"`
}

func toImportBatchJSON(b store.ImportBatch, accountName string) importBatchJSON {
	return importBatchJSON{
		ID: b.ID, AccountID: b.AccountID, AccountName: accountName,
		Filename: b.Filename, RowCount: b.RowCount,
		ImportedCount: b.ImportedCount, SkippedCount: b.SkippedCount,
		CreatedAt: b.CreatedAt,
	}
}

func (api *budgetAPI) listImports(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	batches, err := api.imports.List(r.Context(), user.ID)
	if err != nil {
		apiInternalError(w, "list imports", err)
		return
	}
	out := make([]importBatchJSON, 0, len(batches))
	for _, b := range batches {
		out = append(out, toImportBatchJSON(store.ImportBatch{
			ID: b.ID, AccountID: b.AccountID, Filename: b.Filename,
			RowCount: b.RowCount, ImportedCount: b.ImportedCount,
			SkippedCount: b.SkippedCount, CreatedAt: b.CreatedAt,
		}, b.AccountName))
	}
	writeJSON(w, http.StatusOK, map[string]any{"imports": out})
}

func (api *budgetAPI) revertImport(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	id, ok := pathUUID(r, "id")
	if !ok {
		apiError(w, http.StatusNotFound, "not_found", "import not found")
		return
	}
	switch err := api.imports.Delete(r.Context(), user.ID, id); {
	case errors.Is(err, store.ErrNotFound):
		apiError(w, http.StatusNotFound, "not_found", "import not found")
	case err != nil:
		apiInternalError(w, "revert import", err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func fileSHA(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
