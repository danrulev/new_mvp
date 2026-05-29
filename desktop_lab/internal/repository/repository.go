package repository

import (
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Repository struct{}

func NewRepository(db *sqlx.DB, log *zap.Logger) *Repository {
	return &Repository{}
}
