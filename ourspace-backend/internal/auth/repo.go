package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrAPIKeyNotFound = errors.New("api key not found")
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

func (r *PostgresRepository) FindUserLoginDetails(ctx context.Context, username string) (*LoginDetails, error) {
	var result LoginDetails

	err := r.db.QueryRowContext(
		ctx, `
		select
			members.id,
			members_auth.username,
			members_auth.password_hash,
			members.name
		from members_auth
		inner join members on members.id = members_auth.id
		where
			username = $1
	`, username).Scan(
		&result.ID,
		&result.Username,
		&result.PasswordHash,
		&result.FullName,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w", ErrUserNotFound)
	}

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *PostgresRepository) UpdateHash(ctx context.Context, username, passwordHash string) error {
	_, err := r.db.ExecContext(ctx, `
		update members_auth
		set password_hash = $2
		where username = $1
	`, username, passwordHash)

	return err
}

func (r *PostgresRepository) FindAPIKey(ctx context.Context, apiKey string) (*APIKeyDetails, error) {
	var (
		result   APIKeyDetails
		memberID sql.Null[string]
	)

	err := r.db.QueryRowContext(ctx, `
		select
			id,
			member_id
		from api_keys
		where
			api_key = $1
	`, apiKey).Scan(
		&result.ID,
		&memberID,
	)
	if err != nil {
		return nil, err
	}

	result.MemberID = memberID.V

	return &result, nil
}
