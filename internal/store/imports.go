package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ctrl-research/getbud/internal/store/sqlcgen"
)

type (
	ImportBatch    = sqlcgen.ImportBatch
	ImportBatchRow = sqlcgen.ListImportBatchesRow
)

type Imports struct {
	pool *pgxpool.Pool
	q    *sqlcgen.Queries
}

func NewImports(pool *pgxpool.Pool) *Imports {
	return &Imports{pool: pool, q: sqlcgen.New(pool)}
}

// Commit records the batch and inserts its transactions in one transaction;
// either the whole import lands or none of it does. Rows get their dedup
// hash computed here.
func (s *Imports) Commit(ctx context.Context, userID, accountID uuid.UUID, filename string, mapping []byte, rowCount, skippedCount int, rows []sqlcgen.CreateTransactionParams) (ImportBatch, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ImportBatch{}, err
	}
	defer tx.Rollback(ctx)

	q := s.q.WithTx(tx)
	batch, err := q.CreateImportBatch(ctx, sqlcgen.CreateImportBatchParams{
		UserID: userID, AccountID: accountID, Filename: filename, Mapping: mapping,
		RowCount: int32(rowCount), ImportedCount: int32(len(rows)), SkippedCount: int32(skippedCount),
	})
	if err != nil {
		return ImportBatch{}, translate(err)
	}
	for _, row := range rows {
		row.UserID = userID
		row.AccountID = accountID
		row.ImportBatchID = &batch.ID
		row.DedupHash = DedupHash(row.Date, row.AmountCents, row.Payee)
		if _, err := q.CreateTransaction(ctx, row); err != nil {
			return ImportBatch{}, translate(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ImportBatch{}, err
	}
	return batch, nil
}

func (s *Imports) List(ctx context.Context, userID uuid.UUID) ([]ImportBatchRow, error) {
	batches, err := s.q.ListImportBatches(ctx, userID)
	if batches == nil {
		batches = []ImportBatchRow{}
	}
	return batches, err
}

// LatestMapping returns the column mapping from the account's most recent
// import, or nil when the account has never been imported into.
func (s *Imports) LatestMapping(ctx context.Context, userID, accountID uuid.UUID) ([]byte, error) {
	mapping, err := s.q.LatestMappingForAccount(ctx, sqlcgen.LatestMappingForAccountParams{UserID: userID, AccountID: accountID})
	if err != nil {
		err = translate(err)
		if err == ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return mapping, nil
}

// Delete reverts an import; the FK cascade removes its transactions.
func (s *Imports) Delete(ctx context.Context, userID, id uuid.UUID) error {
	n, err := s.q.DeleteImportBatch(ctx, sqlcgen.DeleteImportBatchParams{UserID: userID, ID: id})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
