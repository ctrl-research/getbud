package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ctrl-research/getbud/internal/store/sqlcgen"
)

type (
	Account     = sqlcgen.Account
	AccountType = sqlcgen.AccountType
	// AccountWithBalance carries the account plus its derived balance
	// (latest snapshot for investment types, transaction sum for cash).
	AccountWithBalance = sqlcgen.ListAccountsRow
)

// InvestmentTypes are the account types whose balance comes from snapshots
// rather than transactions. The registered subset also tracks contribution
// room.
var InvestmentTypes = map[AccountType]bool{
	sqlcgen.AccountTypeRrsp:          true,
	sqlcgen.AccountTypeTfsa:          true,
	sqlcgen.AccountTypeFhsa:          true,
	sqlcgen.AccountTypeNonRegistered: true,
}

type Accounts struct {
	q *sqlcgen.Queries
}

func NewAccounts(pool *pgxpool.Pool) *Accounts {
	return &Accounts{q: sqlcgen.New(pool)}
}

func (s *Accounts) Create(ctx context.Context, userID uuid.UUID, name string, typ AccountType, currency, institution string, openingBalanceCents int64) (Account, error) {
	a, err := s.q.CreateAccount(ctx, sqlcgen.CreateAccountParams{
		UserID: userID, Name: name, Type: typ, Currency: currency,
		Institution: institution, OpeningBalanceCents: openingBalanceCents,
	})
	return a, translate(err)
}

func (s *Accounts) List(ctx context.Context, userID uuid.UUID) ([]AccountWithBalance, error) {
	accounts, err := s.q.ListAccounts(ctx, userID)
	if accounts == nil {
		accounts = []AccountWithBalance{}
	}
	return accounts, err
}

func (s *Accounts) ByID(ctx context.Context, userID, id uuid.UUID) (Account, error) {
	a, err := s.q.AccountByID(ctx, sqlcgen.AccountByIDParams{UserID: userID, ID: id})
	return a, translate(err)
}

func (s *Accounts) Update(ctx context.Context, userID uuid.UUID, a Account) (Account, error) {
	out, err := s.q.UpdateAccount(ctx, sqlcgen.UpdateAccountParams{
		UserID: userID, ID: a.ID, Name: a.Name, Type: a.Type, Currency: a.Currency,
		Institution: a.Institution, OpeningBalanceCents: a.OpeningBalanceCents,
		IsArchived: a.IsArchived,
	})
	return out, translate(err)
}

func (s *Accounts) Delete(ctx context.Context, userID, id uuid.UUID) error {
	n, err := s.q.DeleteAccount(ctx, sqlcgen.DeleteAccountParams{UserID: userID, ID: id})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// TransactionCount reports how many transactions an account has (delete is
// only allowed when empty; otherwise archive).
func (s *Accounts) TransactionCount(ctx context.Context, userID, id uuid.UUID) (int64, error) {
	return s.q.CountAccountTransactions(ctx, sqlcgen.CountAccountTransactionsParams{UserID: userID, AccountID: id})
}
