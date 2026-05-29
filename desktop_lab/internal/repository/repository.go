package repository

import (
	"desktop_lab/internal/repository/mysql_repo"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Repository struct {
	*mysql_repo.Repository
}

func NewRepository(db *sqlx.DB, log *zap.Logger) *Repository {
	return &Repository{
		Repository: mysql_repo.NewRepository(db, log),
	}
}
