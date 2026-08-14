package mysql_repo

import (
	"context"
	"desktop_lab/internal/models"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type TokenRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewTokenRepo(db *sqlx.DB, log *zap.Logger) *TokenRepo {
	return &TokenRepo{db: db, log: log}
}

func (r *TokenRepo) Create(ctx context.Context, token models.Token) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	log := logQuery(ctx, r.log, "INSERT", "tokens",
		zap.String("user_id", token.UserID),
		zap.Time("expires_at", token.ExpiresAt.ToTime()),
	)
	log.Debug("creating new token")

	nowStr := time.Now().UTC().Format(timeLayout)
	expiresStr := token.ExpiresAt.ToTime().Format(timeLayout)

	_, err := r.db.ExecContext(ctx,
		"INSERT INTO tokens (id, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)",
		token.ID, token.UserID, expiresStr, nowStr,
	)
	if err != nil {
		log.Error("failed to create token", zap.Error(err))
		return fmt.Errorf("failed to create token: %w", err)
	}

	log.Info("token created successfully")
	return nil
}

func (r *TokenRepo) TokenByID(ctx context.Context, id string) (models.Token, error) {
	if err := checkContext(ctx); err != nil {
		return models.Token{}, err
	}

	log := logQuery(ctx, r.log, "SELECT", "tokens", zap.String("id", id))
	log.Debug("fetching token by id")

	var token models.Token
	err := r.db.GetContext(ctx, &token,
		"SELECT id, user_id, expires_at, created_at FROM tokens WHERE id = ?", id)
	if err != nil {
		log.Debug("token not found", zap.Error(err))
		return models.Token{}, err
	}

	log.Debug("token retrieved successfully")
	return token, nil
}

func (r *TokenRepo) Delete(ctx context.Context, id string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	log := logQuery(ctx, r.log, "DELETE", "tokens", zap.String("id", id))
	log.Debug("deleting token by id")

	result, err := r.db.ExecContext(ctx, "DELETE FROM tokens WHERE id = ?", id)
	if err != nil {
		log.Error("failed to delete token", zap.Error(err))
		return fmt.Errorf("failed to delete token: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Info("token deleted successfully", zap.Int64("rows_affected", rowsAffected))
	return nil
}
