package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ctrl-research/getbud/internal/store/sqlcgen"
)

type (
	ReportSummary     = sqlcgen.ReportSummaryRow
	CategoryTotalsRow = sqlcgen.ReportCategoryTotalsRow
	TrendsRow         = sqlcgen.ReportTrendsRow
	NetWorthRow       = sqlcgen.ReportNetWorthRow
)

type Reports struct {
	q *sqlcgen.Queries
}

func NewReports(pool *pgxpool.Pool) *Reports {
	return &Reports{q: sqlcgen.New(pool)}
}

func (s *Reports) Summary(ctx context.Context, userID uuid.UUID, from, to time.Time) (ReportSummary, error) {
	return s.q.ReportSummary(ctx, sqlcgen.ReportSummaryParams{UserID: userID, Date: from, Date_2: to})
}

func (s *Reports) CategoryTotals(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]CategoryTotalsRow, error) {
	rows, err := s.q.ReportCategoryTotals(ctx, sqlcgen.ReportCategoryTotalsParams{UserID: userID, Date: from, Date_2: to})
	if rows == nil {
		rows = []CategoryTotalsRow{}
	}
	return rows, err
}

func (s *Reports) Trends(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]TrendsRow, error) {
	rows, err := s.q.ReportTrends(ctx, sqlcgen.ReportTrendsParams{UserID: userID, Date: from, Date_2: to})
	if rows == nil {
		rows = []TrendsRow{}
	}
	return rows, err
}

func (s *Reports) NetWorth(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]NetWorthRow, error) {
	rows, err := s.q.ReportNetWorth(ctx, sqlcgen.ReportNetWorthParams{UserID: userID, Column2: from, Column3: to})
	if rows == nil {
		rows = []NetWorthRow{}
	}
	return rows, err
}
