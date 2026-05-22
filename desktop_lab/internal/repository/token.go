package repository

import (
	"context"
	"desktop_lab/internal/models"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type TokenRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewTokenRepo(db *sqlx.DB, log *zap.Logger) *TokenRepo {
	return &TokenRepo{}
}

func (r *TokenRepo) Create(ctx context.Context, token models.Token) error {
	log := logQuery(ctx, r.log, "INSERT", "tokens", zap.String("user_id", token.UserID))
	log.Info("creating new token")

	_, err := r.db.ExecContext(ctx, "INSERT INTO tokens (id, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}

	log.Info("Token created successfully")
	return nil
}

func (r *TokenRepo) TokenByID(ctx context.Context, id string) (models.Token, error) {
	log := logQuery(ctx, r.log, "SELECT", "tokens",
		zap.String("id", id),
	)
	log.Debug("fetching token by id")

	var token models.Token
	err := r.db.GetContext(ctx, &token, "SELECT id, user_id, expires_at, created_at FROM tokens WHERE id = ?")
	if err != nil {
		return models.Token{}, err
	}

	log.Debug("token retrieved successfully")
	return token, nil
}

func (r *TokenRepo) Delete(ctx context.Context, id string) error {
	log := logQuery(ctx, r.log, "DELETE", "tokens",
		zap.String("id", id),
	)
	log.Debug("deleting token by id")

	_, err := r.db.ExecContext(ctx, "DELETE FROM tokens WHERE id = ?")
	if err != nil {
		return err
	}

	log.Debug("token deleted successfully")
	return nil
}
