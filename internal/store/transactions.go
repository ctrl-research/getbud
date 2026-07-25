package store

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ctrl-research/getbud/internal/store/sqlcgen"
)

type (
	Transaction             = sqlcgen.Transaction
	TransactionRow          = sqlcgen.ListTransactionsRow
	ListParams              = sqlcgen.ListTransactionsParams
	DedupMatch              = sqlcgen.FindByDedupHashesRow
	CreateTransactionParams = sqlcgen.CreateTransactionParams
)

type Transactions struct {
	pool *pgxpool.Pool
	q    *sqlcgen.Queries
}

func NewTransactions(pool *pgxpool.Pool) *Transactions {
	return &Transactions{pool: pool, q: sqlcgen.New(pool)}
}

// DedupHash fingerprints a transaction for soft duplicate detection:
// sha256(date|amount|normalized payee). Payee normalization lowercases,
// strips non-alphanumerics, and collapses whitespace so "TIM HORTONS #1234"
// and "Tim Hortons  1234" collide.
func DedupHash(date time.Time, amountCents int64, payee string) []byte {
	var b strings.Builder
	lastSpace := true
	for _, r := range strings.ToLower(payee) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastSpace = false
		case !lastSpace:
			b.WriteRune(' ')
			lastSpace = true
		}
	}
	normalized := strings.TrimSpace(b.String())
	sum := sha256.Sum256(fmt.Appendf(nil, "%s|%d|%s", date.Format("2006-01-02"), amountCents, normalized))
	return sum[:]
}

func (s *Transactions) Create(ctx context.Context, p CreateTransactionParams) (Transaction, error) {
	p.DedupHash = DedupHash(p.Date, p.AmountCents, p.Payee)
	t, err := s.q.CreateTransaction(ctx, p)
	return t, translate(err)
}

func (s *Transactions) ByID(ctx context.Context, userID, id uuid.UUID) (Transaction, error) {
	t, err := s.q.TransactionByID(ctx, sqlcgen.TransactionByIDParams{UserID: userID, ID: id})
	return t, translate(err)
}

func (s *Transactions) List(ctx context.Context, p ListParams) ([]TransactionRow, error) {
	rows, err := s.q.ListTransactions(ctx, p)
	if rows == nil {
		rows = []TransactionRow{}
	}
	return rows, err
}

func (s *Transactions) Count(ctx context.Context, p ListParams) (int64, error) {
	return s.q.CountTransactions(ctx, sqlcgen.CountTransactionsParams{
		UserID: p.UserID, FromDate: p.FromDate, ToDate: p.ToDate,
		AccountID: p.AccountID, CategoryID: p.CategoryID,
		Uncategorized: p.Uncategorized, Search: p.Search,
	})
}

// Update rewrites the editable fields and recomputes the dedup hash.
func (s *Transactions) Update(ctx context.Context, userID uuid.UUID, t Transaction) (Transaction, error) {
	out, err := s.q.UpdateTransaction(ctx, sqlcgen.UpdateTransactionParams{
		UserID: userID, ID: t.ID, Date: t.Date, AmountCents: t.AmountCents,
		Payee: t.Payee, Notes: t.Notes, CategoryID: t.CategoryID,
		DedupHash: DedupHash(t.Date, t.AmountCents, t.Payee),
	})
	return out, translate(err)
}

// Delete removes a transaction; deleting a transfer leg removes both legs.
func (s *Transactions) Delete(ctx context.Context, userID, id uuid.UUID) error {
	n, err := s.q.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{UserID: userID, ID: id})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateTransfer inserts both legs of a transfer between two of the user's
// accounts atomically. Legs share a transfer_group_id and have no category.
func (s *Transactions) CreateTransfer(ctx context.Context, userID, fromAccount, toAccount uuid.UUID, date time.Time, amountCents int64, notes string) ([]Transaction, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q := s.q.WithTx(tx)
	group := uuid.New()
	legs := make([]Transaction, 0, 2)
	for _, leg := range []struct {
		account uuid.UUID
		amount  int64
	}{
		{fromAccount, -amountCents},
		{toAccount, amountCents},
	} {
		t, err := q.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
			UserID: userID, AccountID: leg.account, Date: date,
			AmountCents: leg.amount, Notes: notes,
			TransferGroupID: &group,
			DedupHash:       DedupHash(date, leg.amount, "transfer"),
		})
		if err != nil {
			return nil, translate(err)
		}
		legs = append(legs, t)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return legs, nil
}

// FindByDedupHashes returns existing transactions in the account whose dedup
// hash matches any of the given hashes (CSV import duplicate warnings).
func (s *Transactions) FindByDedupHashes(ctx context.Context, userID, accountID uuid.UUID, hashes [][]byte) ([]DedupMatch, error) {
	rows, err := s.q.FindByDedupHashes(ctx, sqlcgen.FindByDedupHashesParams{
		UserID: userID, AccountID: accountID, Column3: hashes,
	})
	if rows == nil {
		rows = []DedupMatch{}
	}
	return rows, err
}
