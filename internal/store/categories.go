package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ctrl-research/getbud/internal/store/sqlcgen"
)

type (
	Category     = sqlcgen.Category
	CategoryKind = sqlcgen.CategoryKind
)

const (
	KindIncome  = sqlcgen.CategoryKindIncome
	KindExpense = sqlcgen.CategoryKindExpense
)

// defaultCategories is the starter set inserted for every new user. Colors
// are stable per category (identity, not rank) and drawn from the validated
// chart palette hues; users can change them in Settings.
var defaultCategories = []struct {
	Name  string
	Kind  CategoryKind
	Color string
}{
	{"Salary", KindIncome, "#2a78d6"},
	{"Bonus", KindIncome, "#1baf7a"},
	{"Interest", KindIncome, "#4a3aa7"},
	{"Investment Income", KindIncome, "#008300"},
	{"Gifts", KindIncome, "#e87ba4"},
	{"Other Income", KindIncome, "#898781"},

	{"Rent/Mortgage", KindExpense, "#4a3aa7"},
	{"Groceries", KindExpense, "#008300"},
	{"Restaurants", KindExpense, "#eb6834"},
	{"Transit", KindExpense, "#2a78d6"},
	{"Car", KindExpense, "#1c5cab"},
	{"Utilities", KindExpense, "#eda100"},
	{"Phone & Internet", KindExpense, "#1baf7a"},
	{"Insurance", KindExpense, "#52514e"},
	{"Health", KindExpense, "#e34948"},
	{"Shopping", KindExpense, "#e87ba4"},
	{"Entertainment", KindExpense, "#d95926"},
	{"Travel", KindExpense, "#199e70"},
	{"Subscriptions", KindExpense, "#9085e9"},
	{"Fees", KindExpense, "#898781"},
	{"Gifts & Donations", KindExpense, "#d55181"},
	{"Personal Care", KindExpense, "#c98500"},
	{"Other", KindExpense, "#b3b2ab"},
}

type Categories struct {
	q *sqlcgen.Queries
}

func NewCategories(pool *pgxpool.Pool) *Categories {
	return &Categories{q: sqlcgen.New(pool)}
}

func (s *Categories) Create(ctx context.Context, userID uuid.UUID, name string, kind CategoryKind, color string) (Category, error) {
	c, err := s.q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		UserID: userID, Name: name, Kind: kind, Color: color,
	})
	return c, translate(err)
}

func (s *Categories) List(ctx context.Context, userID uuid.UUID) ([]Category, error) {
	cats, err := s.q.ListCategories(ctx, userID)
	if cats == nil {
		cats = []Category{}
	}
	return cats, err
}

func (s *Categories) ByID(ctx context.Context, userID, id uuid.UUID) (Category, error) {
	c, err := s.q.CategoryByID(ctx, sqlcgen.CategoryByIDParams{UserID: userID, ID: id})
	return c, translate(err)
}

func (s *Categories) Update(ctx context.Context, userID uuid.UUID, c Category) (Category, error) {
	out, err := s.q.UpdateCategory(ctx, sqlcgen.UpdateCategoryParams{
		UserID: userID, ID: c.ID, Name: c.Name, Color: c.Color, IsArchived: c.IsArchived,
	})
	return out, translate(err)
}

func (s *Categories) Delete(ctx context.Context, userID, id uuid.UUID) error {
	n, err := s.q.DeleteCategory(ctx, sqlcgen.DeleteCategoryParams{UserID: userID, ID: id})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Reassign moves all transactions from one category to another before a
// delete ("merge"). Returns the number of transactions moved.
func (s *Categories) Reassign(ctx context.Context, userID, from, to uuid.UUID) (int64, error) {
	return s.q.ReassignCategory(ctx, sqlcgen.ReassignCategoryParams{
		UserID: userID, CategoryID: &from, CategoryID_2: &to,
	})
}
