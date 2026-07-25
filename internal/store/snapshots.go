package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ctrl-research/getbud/internal/store/sqlcgen"
)

type BalanceSnapshot = sqlcgen.BalanceSnapshot

type Snapshots struct {
	q *sqlcgen.Queries
}

func NewSnapshots(pool *pgxpool.Pool) *Snapshots {
	return &Snapshots{q: sqlcgen.New(pool)}
}

// Upsert records the account balance as of a date, replacing any snapshot
// already taken that day. Callers must verify account ownership first.
func (s *Snapshots) Upsert(ctx context.Context, userID, accountID uuid.UUID, asOf time.Time, balanceCents int64) (BalanceSnapshot, error) {
	snap, err := s.q.UpsertSnapshot(ctx, sqlcgen.UpsertSnapshotParams{
		UserID: userID, AccountID: accountID, AsOf: asOf, BalanceCents: balanceCents,
	})
	return snap, translate(err)
}

func (s *Snapshots) List(ctx context.Context, userID, accountID uuid.UUID) ([]BalanceSnapshot, error) {
	snaps, err := s.q.ListSnapshots(ctx, sqlcgen.ListSnapshotsParams{UserID: userID, AccountID: accountID})
	if snaps == nil {
		snaps = []BalanceSnapshot{}
	}
	return snaps, err
}

func (s *Snapshots) Delete(ctx context.Context, userID, id uuid.UUID) error {
	n, err := s.q.DeleteSnapshot(ctx, sqlcgen.DeleteSnapshotParams{UserID: userID, ID: id})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
