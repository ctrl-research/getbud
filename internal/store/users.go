package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ctrl-research/getbud/internal/store/sqlcgen"
)

type (
	User             = sqlcgen.User
	CreateUserParams = sqlcgen.CreateUserParams
)

type Users struct {
	pool *pgxpool.Pool
	q    *sqlcgen.Queries
}

func NewUsers(pool *pgxpool.Pool) *Users {
	return &Users{pool: pool, q: sqlcgen.New(pool)}
}

// Create inserts the user and seeds their default category set in one
// transaction, so every sign-up path (Google, OIDC, local) starts with
// usable categories.
func (s *Users) Create(ctx context.Context, p CreateUserParams) (User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	q := s.q.WithTx(tx)
	u, err := q.CreateUser(ctx, p)
	if err != nil {
		return User{}, translate(err)
	}

	seed := sqlcgen.SeedDefaultCategoriesParams{UserID: u.ID}
	for _, c := range defaultCategories {
		seed.Names = append(seed.Names, c.Name)
		seed.Kinds = append(seed.Kinds, string(c.Kind))
		seed.Colors = append(seed.Colors, c.Color)
	}
	if err := q.SeedDefaultCategories(ctx, seed); err != nil {
		return User{}, translate(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Users) ByID(ctx context.Context, id uuid.UUID) (User, error) {
	u, err := s.q.UserByID(ctx, id)
	return u, translate(err)
}

func (s *Users) ByEmail(ctx context.Context, email string) (User, error) {
	u, err := s.q.UserByEmail(ctx, email)
	return u, translate(err)
}

func (s *Users) ByGoogleSub(ctx context.Context, sub string) (User, error) {
	u, err := s.q.UserByGoogleSub(ctx, &sub)
	return u, translate(err)
}

// UpdateProfile refreshes the fields Google reports on each sign-in.
func (s *Users) UpdateProfile(ctx context.Context, id uuid.UUID, displayName string, avatarURL *string) (User, error) {
	u, err := s.q.UpdateUserProfile(ctx, sqlcgen.UpdateUserProfileParams{
		ID: id, DisplayName: displayName, AvatarURL: avatarURL,
	})
	return u, translate(err)
}

// LinkGoogle attaches a Google identity to an existing account (matched by
// email at sign-in) and refreshes the profile fields Google reports.
func (s *Users) LinkGoogle(ctx context.Context, id uuid.UUID, googleSub string, displayName string, avatarURL *string) (User, error) {
	u, err := s.q.LinkGoogle(ctx, sqlcgen.LinkGoogleParams{
		ID: id, GoogleSub: &googleSub, DisplayName: displayName, AvatarURL: avatarURL,
	})
	return u, translate(err)
}

// SetPassword replaces the user's password hash (used by the seed command).
func (s *Users) SetPassword(ctx context.Context, id uuid.UUID, passwordHash string) (User, error) {
	u, err := s.q.SetPassword(ctx, sqlcgen.SetPasswordParams{ID: id, PasswordHash: &passwordHash})
	return u, translate(err)
}

func (s *Users) Count(ctx context.Context) (int64, error) {
	return s.q.CountUsers(ctx)
}

// ByOIDCSub finds the user linked to a generic OIDC provider subject.
func (s *Users) ByOIDCSub(ctx context.Context, sub string) (User, error) {
	u, err := s.q.UserByOIDCSub(ctx, &sub)
	return u, translate(err)
}

// LinkOIDC attaches a generic OIDC subject to an existing account and
// refreshes the profile from the provider's claims.
func (s *Users) LinkOIDC(ctx context.Context, id uuid.UUID, sub string, displayName string, avatarURL *string) (User, error) {
	u, err := s.q.LinkOIDC(ctx, sqlcgen.LinkOIDCParams{
		ID: id, OidcSub: &sub, DisplayName: displayName, AvatarURL: avatarURL,
	})
	return u, translate(err)
}
