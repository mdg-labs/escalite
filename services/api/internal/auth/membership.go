package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/internal/db"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNoActiveMembership = errors.New("no active organization membership")
	ErrForbiddenOrg       = errors.New("organization access denied")
)

// AuthenticateAccount verifies email/password credentials and returns the account.
func AuthenticateAccount(ctx context.Context, q db.Querier, email, password string) (db.Account, error) {
	account, err := q.GetAccountByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Account{}, ErrInvalidCredentials
		}
		return db.Account{}, err
	}

	if !account.PasswordHash.Valid || account.PasswordHash.String == "" {
		return db.Account{}, ErrInvalidCredentials
	}

	match, err := VerifyPassword(password, account.PasswordHash.String)
	if err != nil {
		return db.Account{}, err
	}
	if !match {
		return db.Account{}, ErrInvalidCredentials
	}

	return account, nil
}

// LoginMembership returns the org membership to use after authentication.
func LoginMembership(ctx context.Context, q db.Querier, accountID uuid.UUID) (db.User, error) {
	user, err := q.GetLoginMembershipByAccountID(ctx, accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.User{}, ErrNoActiveMembership
		}
		return db.User{}, err
	}
	if user.DeprovisionedAt.Valid {
		return db.User{}, ErrNoActiveMembership
	}
	return user, nil
}

// SwitchOrganization updates the session to the requested org membership.
func SwitchOrganization(
	ctx context.Context,
	q db.Querier,
	session db.Session,
	accountID uuid.UUID,
	organizationID uuid.UUID,
) (db.User, db.Session, error) {
	user, err := q.GetUserByAccountAndOrganization(ctx, db.GetUserByAccountAndOrganizationParams{
		AccountID:      accountID,
		OrganizationID: organizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.User{}, db.Session{}, ErrForbiddenOrg
		}
		return db.User{}, db.Session{}, err
	}

	updatedSession, err := q.UpdateSessionOrganization(ctx, db.UpdateSessionOrganizationParams{
		ID:             session.ID,
		UserID:         user.ID,
		OrganizationID: organizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.User{}, db.Session{}, ErrForbiddenOrg
		}
		return db.User{}, db.Session{}, err
	}

	return user, updatedSession, nil
}

// AccountIDFromUser loads the account ID for an org-scoped user row.
func AccountIDFromUser(ctx context.Context, q db.Querier, user db.User) (uuid.UUID, error) {
	if user.AccountID != uuid.Nil {
		return user.AccountID, nil
	}

	loaded, err := q.GetUserByID(ctx, db.GetUserByIDParams{
		ID:             user.ID,
		OrganizationID: user.OrganizationID,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return loaded.AccountID, nil
}

// PasswordHashParam converts a password hash string for account storage.
func PasswordHashParam(passwordHash string) pgtype.Text {
	return pgtype.Text{String: passwordHash, Valid: passwordHash != ""}
}

// FindOrCreateAccountByEmail returns an existing account or creates one without a password.
func FindOrCreateAccountByEmail(ctx context.Context, q db.Querier, email string) (db.Account, error) {
	account, err := q.GetAccountByEmail(ctx, email)
	if err == nil {
		return account, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return db.Account{}, err
	}

	accountID := uuid.Must(uuid.NewV7())
	return q.CreateAccount(ctx, db.CreateAccountParams{
		ID:           accountID,
		Email:        email,
		PasswordHash: pgtype.Text{},
	})
}

// EnsureOrgMembership returns the org-scoped user row for an account, creating it when missing.
func EnsureOrgMembership(
	ctx context.Context,
	q db.Querier,
	account db.Account,
	organizationID uuid.UUID,
	role string,
) (db.User, error) {
	user, err := q.GetUserByAccountAndOrganization(ctx, db.GetUserByAccountAndOrganizationParams{
		AccountID:      account.ID,
		OrganizationID: organizationID,
	})
	if err == nil {
		if user.DeprovisionedAt.Valid {
			return db.User{}, ErrNoActiveMembership
		}
		return user, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, err
	}

	userID := uuid.Must(uuid.NewV7())
	return q.CreateUser(ctx, db.CreateUserParams{
		ID:             userID,
		AccountID:      account.ID,
		OrganizationID: organizationID,
		Email:          account.Email,
		Role:           role,
	})
}
