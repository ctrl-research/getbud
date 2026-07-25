package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ctrl-research/getbud/internal/store/sqlcgen"
)

type (
	ContributionRoom   = sqlcgen.ContributionRoom
	ContributionTotals = sqlcgen.ContributionTotalsRow
)

type Contributions struct {
	q *sqlcgen.Queries
}

func NewContributions(pool *pgxpool.Pool) *Contributions {
	return &Contributions{q: sqlcgen.New(pool)}
}

func (s *Contributions) UpsertRoom(ctx context.Context, userID uuid.UUID, typ AccountType, taxYear int, roomCents int64, notes string) (ContributionRoom, error) {
	room, err := s.q.UpsertContributionRoom(ctx, sqlcgen.UpsertContributionRoomParams{
		UserID: userID, AccountType: typ, TaxYear: int32(taxYear), RoomCents: roomCents, Notes: notes,
	})
	return room, translate(err)
}

func (s *Contributions) ListRoom(ctx context.Context, userID uuid.UUID, taxYear int) ([]ContributionRoom, error) {
	rooms, err := s.q.ListContributionRoom(ctx, sqlcgen.ListContributionRoomParams{UserID: userID, TaxYear: int32(taxYear)})
	if rooms == nil {
		rooms = []ContributionRoom{}
	}
	return rooms, err
}

// Totals returns derived calendar-year contributions and withdrawals per
// registered account type (rrsp/tfsa/fhsa); types with no activity are absent.
func (s *Contributions) Totals(ctx context.Context, userID uuid.UUID, year int) ([]ContributionTotals, error) {
	totals, err := s.q.ContributionTotals(ctx, sqlcgen.ContributionTotalsParams{UserID: userID, Year: int32(year)})
	if totals == nil {
		totals = []ContributionTotals{}
	}
	return totals, err
}
