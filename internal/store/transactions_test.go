package store_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ctrl-research/getbud/internal/store"
	"github.com/ctrl-research/getbud/internal/store/sqlcgen"
	"github.com/ctrl-research/getbud/internal/store/storetest"
)

func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func createUser(t *testing.T, pool *pgxpool.Pool, email string) store.User {
	t.Helper()
	hash := "x"
	u, err := store.NewUsers(pool).Create(context.Background(), store.CreateUserParams{
		Email: email, DisplayName: "Test", PasswordHash: &hash,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

func createAccount(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, name string, typ store.AccountType) store.Account {
	t.Helper()
	a, err := store.NewAccounts(pool).Create(context.Background(), userID, name, typ, "CAD", "", 0)
	if err != nil {
		t.Fatalf("create account %s: %v", name, err)
	}
	return a
}

func TestUserCreateSeedsCategories(t *testing.T) {
	pool := storetest.Pool(t)
	u := createUser(t, pool, "seed@test.local")

	cats, err := store.NewCategories(pool).List(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(cats) < 20 {
		t.Fatalf("expected default categories seeded, got %d", len(cats))
	}
	var income, expense int
	for _, c := range cats {
		switch c.Kind {
		case store.KindIncome:
			income++
		case store.KindExpense:
			expense++
		}
	}
	if income == 0 || expense == 0 {
		t.Fatalf("expected both kinds seeded, got income=%d expense=%d", income, expense)
	}
}

func TestTransferCreateAndDelete(t *testing.T) {
	pool := storetest.Pool(t)
	ctx := context.Background()
	u := createUser(t, pool, "transfer@test.local")
	chequing := createAccount(t, pool, u.ID, "Chequing", sqlcgen.AccountTypeChequing)
	tfsa := createAccount(t, pool, u.ID, "TFSA", sqlcgen.AccountTypeTfsa)

	txns := store.NewTransactions(pool)
	legs, err := txns.CreateTransfer(ctx, u.ID, chequing.ID, tfsa.ID, date("2026-01-15"), 50_000, "monthly savings")
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	if len(legs) != 2 {
		t.Fatalf("expected 2 legs, got %d", len(legs))
	}
	if legs[0].AmountCents != -50_000 || legs[1].AmountCents != 50_000 {
		t.Fatalf("unexpected amounts: %d, %d", legs[0].AmountCents, legs[1].AmountCents)
	}
	if legs[0].TransferGroupID == nil || legs[1].TransferGroupID == nil || *legs[0].TransferGroupID != *legs[1].TransferGroupID {
		t.Fatal("legs must share a transfer group")
	}
	if legs[0].CategoryID != nil || legs[1].CategoryID != nil {
		t.Fatal("transfer legs must be uncategorized")
	}

	// Deleting one leg removes both.
	if err := txns.Delete(ctx, u.ID, legs[0].ID); err != nil {
		t.Fatalf("delete transfer leg: %v", err)
	}
	if _, err := txns.ByID(ctx, u.ID, legs[1].ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected peer leg deleted, got err=%v", err)
	}
}

func TestUserIsolation(t *testing.T) {
	pool := storetest.Pool(t)
	ctx := context.Background()
	alice := createUser(t, pool, "alice@test.local")
	bob := createUser(t, pool, "bob@test.local")
	aliceAcct := createAccount(t, pool, alice.ID, "Alice Chequing", sqlcgen.AccountTypeChequing)

	txns := store.NewTransactions(pool)
	created, err := txns.Create(ctx, store.CreateTransactionParams{
		UserID: alice.ID, AccountID: aliceAcct.ID, Date: date("2026-02-01"),
		AmountCents: -1234, Payee: "Coffee",
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	// Bob cannot see or touch Alice's data.
	if _, err := txns.ByID(ctx, bob.ID, created.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-user read, got %v", err)
	}
	if err := txns.Delete(ctx, bob.ID, created.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-user delete, got %v", err)
	}
	rows, err := txns.List(ctx, store.ListParams{UserID: bob.ID, RowLimit: 100})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("bob should see no transactions, got %d", len(rows))
	}
	if _, err := store.NewAccounts(pool).ByID(ctx, bob.ID, aliceAcct.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-user account read, got %v", err)
	}
}

func TestDedupHashNormalization(t *testing.T) {
	d := date("2026-03-01")
	a := store.DedupHash(d, -1500, "TIM HORTONS #1234")
	b := store.DedupHash(d, -1500, "tim hortons  1234")
	if !bytes.Equal(a, b) {
		t.Fatal("normalized payees should hash identically")
	}
	c := store.DedupHash(d, -1500, "Starbucks")
	if bytes.Equal(a, c) {
		t.Fatal("different payees must not collide")
	}
	e := store.DedupHash(date("2026-03-02"), -1500, "TIM HORTONS #1234")
	if bytes.Equal(a, e) {
		t.Fatal("different dates must not collide")
	}
}

func TestContributionTotals(t *testing.T) {
	pool := storetest.Pool(t)
	ctx := context.Background()
	u := createUser(t, pool, "contrib@test.local")
	chequing := createAccount(t, pool, u.ID, "Chequing", sqlcgen.AccountTypeChequing)
	tfsa1 := createAccount(t, pool, u.ID, "TFSA WS", sqlcgen.AccountTypeTfsa)
	tfsa2 := createAccount(t, pool, u.ID, "TFSA Bank", sqlcgen.AccountTypeTfsa)
	rrsp := createAccount(t, pool, u.ID, "RRSP", sqlcgen.AccountTypeRrsp)

	txns := store.NewTransactions(pool)
	// Cross-type transfer: counts as a TFSA contribution.
	if _, err := txns.CreateTransfer(ctx, u.ID, chequing.ID, tfsa1.ID, date("2026-01-10"), 300_000, ""); err != nil {
		t.Fatal(err)
	}
	// Same-type transfer: moving TFSA money is NOT a new contribution.
	if _, err := txns.CreateTransfer(ctx, u.ID, tfsa1.ID, tfsa2.ID, date("2026-02-10"), 100_000, ""); err != nil {
		t.Fatal(err)
	}
	// Withdrawal to chequing: counts as a TFSA withdrawal.
	if _, err := txns.CreateTransfer(ctx, u.ID, tfsa2.ID, chequing.ID, date("2026-03-10"), 25_000, ""); err != nil {
		t.Fatal(err)
	}
	// Direct RRSP deposit (e.g. employer plan) recorded as a plain transaction.
	if _, err := txns.Create(ctx, store.CreateTransactionParams{
		UserID: u.ID, AccountID: rrsp.ID, Date: date("2026-04-01"),
		AmountCents: 200_000, Payee: "Employer RRSP",
	}); err != nil {
		t.Fatal(err)
	}
	// Previous year's contribution must not count in 2026.
	if _, err := txns.CreateTransfer(ctx, u.ID, chequing.ID, tfsa1.ID, date("2025-12-15"), 999_00, ""); err != nil {
		t.Fatal(err)
	}

	totals, err := store.NewContributions(pool).Totals(ctx, u.ID, 2026)
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	byType := map[store.AccountType]store.ContributionTotals{}
	for _, tt := range totals {
		byType[tt.AccountType] = tt
	}
	tfsaTotals := byType[sqlcgen.AccountTypeTfsa]
	if tfsaTotals.ContributedCents != 300_000 {
		t.Errorf("tfsa contributed = %d, want 300000 (same-type transfer must be excluded)", tfsaTotals.ContributedCents)
	}
	if tfsaTotals.WithdrawnCents != 25_000 {
		t.Errorf("tfsa withdrawn = %d, want 25000", tfsaTotals.WithdrawnCents)
	}
	rrspTotals := byType[sqlcgen.AccountTypeRrsp]
	if rrspTotals.ContributedCents != 200_000 {
		t.Errorf("rrsp contributed = %d, want 200000", rrspTotals.ContributedCents)
	}
}

func TestAccountBalances(t *testing.T) {
	pool := storetest.Pool(t)
	ctx := context.Background()
	u := createUser(t, pool, "balances@test.local")
	accounts := store.NewAccounts(pool)

	chequing, err := accounts.Create(ctx, u.ID, "Chequing", sqlcgen.AccountTypeChequing, "CAD", "", 100_000)
	if err != nil {
		t.Fatal(err)
	}
	rrsp := createAccount(t, pool, u.ID, "RRSP", sqlcgen.AccountTypeRrsp)

	txns := store.NewTransactions(pool)
	if _, err := txns.Create(ctx, store.CreateTransactionParams{
		UserID: u.ID, AccountID: chequing.ID, Date: date("2026-01-05"), AmountCents: -30_000, Payee: "Rent",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.NewSnapshots(pool).Upsert(ctx, u.ID, rrsp.ID, date("2026-01-31"), 555_000); err != nil {
		t.Fatal(err)
	}

	list, err := accounts.List(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]int64{}
	for _, a := range list {
		byName[a.Account.Name] = a.BalanceCents
	}
	if byName["Chequing"] != 70_000 {
		t.Errorf("chequing balance = %d, want 70000 (opening + txns)", byName["Chequing"])
	}
	if byName["RRSP"] != 555_000 {
		t.Errorf("rrsp balance = %d, want 555000 (latest snapshot)", byName["RRSP"])
	}
}
