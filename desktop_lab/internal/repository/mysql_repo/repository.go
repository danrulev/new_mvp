package mysql_repo

import (
	"context"
	contextkeys "desktop_lab/internal/contextKey"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Repository struct {
	Dimension    *DimensionRepo
	Group        *ExperimentGroupRepo
	Material     *MaterialRepo
	Organization *OrganizationRepo
	Protocol     *ProtocolRepo
	Sample       *SampleRepo
	Standard     *StandardRepo
	Token        *TokenRepo
	User         *UserRepo
}

func NewRepository(db *sqlx.DB, log *zap.Logger) *Repository {
	return &Repository{
		Dimension:    NewDimensionRepo(db, log),
		Group:        NewExperimentGroupRepo(db, log),
		Material:     NewMaterialRepo(db, log),
		Organization: NewOrganizationRepo(db, log),
		Protocol:     NewProtocolRepo(db, log),
		Sample:       NewSampleRepo(db, log),
		Standard:     NewStandardRepo(db, log),
		Token:        NewTokenRepo(db, log),
		User:         NewUserRepo(db, log),
	}
}

func loggerWith(ctx context.Context, log *zap.Logger, fields ...zap.Field) *zap.Logger {
	requestID := ctx.Value(contextkeys.RequestIDKey)
	return log.With(append([]zap.Field{zap.String("request_id", requestID.(string))}, fields...)...)
}

func logQuery(ctx context.Context, log *zap.Logger, operation, table string, fields ...zap.Field) *zap.Logger {
	return loggerWith(ctx, log, append([]zap.Field{
		zap.String("repo", "DimensionRepo"),
		zap.String("operation", operation),
		zap.String("table", table),
	}, fields...)...)
}
