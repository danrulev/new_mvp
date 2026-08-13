package mysql_repo

import (
	"context"
	"database/sql"
	"desktop_lab/internal/models"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type ExperimentGroupRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewExperimentGroupRepo(db *sqlx.DB, log *zap.Logger) *ExperimentGroupRepo {
	return &ExperimentGroupRepo{db: db, log: log}
}

func (r *ExperimentGroupRepo) Create(ctx context.Context, g models.ExperimentGroup) error {
	log := logQuery(ctx, r.log, "INSERT", "experiment_groups",
		zap.String("group_id", g.ID),
		zap.String("group_name", g.Name),
		zap.String("material_id", g.MaterialID),
		zap.String("project_name", g.ProjectName),
		zap.String("object_type", g.ObjectType),
		zap.String("customer", g.Customer),
		zap.String("contract_number", g.ContractNumber),
		zap.String("status", g.Status),
		zap.String("responsible_person_id", g.ResponsiblePersonID),
		zap.String("location", g.Location),
		zap.String("description", g.Description),
	)
	log.Debug("creating new experiment group")

	nowUTC := time.Now().UTC()
	nowStr := nowUTC.Format(timeLayout)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO experiment_groups (id, name, material_id, project_name, object_type, customer, contract_number, status, responsible_person_id, location, description, created_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		g.ID, g.Name, g.MaterialID, g.ProjectName, g.ObjectType, g.Customer, g.ContractNumber, g.Status, g.ResponsiblePersonID, g.Location, g.Description, nowStr,
	)
	if err != nil {
		return fmt.Errorf("failed to create experiment group: %w", err)
	}
	r.log.Info("Experiment group created", zap.String("id", g.ID))
	return nil
}

func (r *ExperimentGroupRepo) GetByID(ctx context.Context, id string) (models.ExperimentGroup, error) {
	log := logQuery(ctx, r.log, "SELECT", "experiment_groups",
		zap.String("group_id", id))
	log.Debug("fetching experiment group by ID")

	var createdAt, updatedAt sql.NullString
	var g models.ExperimentGroup
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, material_id, project_name, object_type, customer, contract_number, status, responsible_person_id, location, description, created_at, updated_at 
		 FROM experiment_groups WHERE id = ?`, id,
	).Scan(&g.ID, &g.Name, &g.MaterialID, &g.ProjectName, &g.ObjectType, &g.Customer, &g.ContractNumber, &g.Status, &g.ResponsiblePersonID, &g.Location, &g.Description, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		log.Debug("group not found")
		return models.ExperimentGroup{}, nil
	}
	if err != nil {
		return models.ExperimentGroup{}, err
	}

	g.CreatedAt, err = parseTime(createdAt.String)
	if err != nil {
		r.log.Warn("failed to parse created_at for group", zap.String("id", id), zap.Error(err))
		g.CreatedAt = time.Now() // Fallback
	}

	if updatedAt.Valid {
		g.UpdatedAt, err = parseTime(updatedAt.String)
		if err != nil {
			r.log.Warn("failed to parse updated_at for group", zap.String("id", id), zap.Error(err))
		}
	}

	log.Debug("group retrieved successfully",
		zap.String("group_name", g.Name))
	return g, nil
}

func (r *ExperimentGroupRepo) GetList(ctx context.Context, p models.GroupListFilter) ([]models.ExperimentGroup, int64, error) {
	log := logQuery(ctx, r.log, "SELECT", "experiment_groups",
		zap.Int64("limit", p.Limit),
		zap.Int64("offset", p.Offset))
	log.Debug("fetching paginated groups list")

	var (
		filterFields []string
		filterArgs   []interface{}
	)

	if p.Name != nil {
		filterFields = append(filterFields, "name LIKE ?")
		filterArgs = append(filterArgs, "%"+*p.Name+"%")
	}

	if p.Location != nil {
		filterFields = append(filterFields, "location LIKE ?")
		filterArgs = append(filterArgs, "%"+*p.Location+"%")
	}

	if p.ProjectName != nil {
		filterFields = append(filterFields, "project_name LIKE ?")
		filterArgs = append(filterArgs, "%"+*p.ProjectName+"%")
	}

	if p.ObjectType != nil {
		filterFields = append(filterFields, "object_type LIKE ?")
		filterArgs = append(filterArgs, "%"+*p.ObjectType+"%")
	}

	if p.Customer != nil {
		filterFields = append(filterFields, "customer LIKE ?")
		filterArgs = append(filterArgs, "%"+*p.Customer+"%")
	}

	if p.Status != nil {
		filterFields = append(filterFields, "status = ?")
		filterArgs = append(filterArgs, *p.Status)
	}

	if p.Material != nil && *p.Material != "" {
		filterFields = append(filterFields, "material_id IN (SELECT id FROM materials WHERE name LIKE ?)")
		filterArgs = append(filterArgs, "%"+*p.Material+"%")
	}

	var whereClause string
	if len(filterFields) > 0 {
		whereClause = " WHERE " + strings.Join(filterFields, " AND ")
	}

	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM experiment_groups`+whereClause, filterArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []models.ExperimentGroup{}, 0, nil
	}

	selectArgs := append([]interface{}{}, filterArgs...)
	selectArgs = append(selectArgs, p.Limit, p.Offset)

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, material_id, project_name, object_type, customer, contract_number, status, responsible_person_id, location, description, created_at, updated_at 
		 FROM experiment_groups`+whereClause+`
		 ORDER BY created_at DESC 
		 LIMIT ? OFFSET ?`, selectArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	groups := make([]models.ExperimentGroup, 0, p.Limit)
	for rows.Next() {
		var g models.ExperimentGroup
		var createdAt, updatedAt sql.NullString
		err := rows.Scan(&g.ID, &g.Name, &g.MaterialID, &g.ProjectName, &g.ObjectType, &g.Customer, &g.ContractNumber, &g.Status, &g.ResponsiblePersonID, &g.Location, &g.Description, &createdAt, &updatedAt)
		if err != nil {
			return nil, 0, err
		}

		parsedTime, err := parseTime(createdAt.String)
		if err != nil {
			r.log.Warn("failed to parse created_at in list", zap.Error(err))
			parsedTime = time.Now()
		}
		g.CreatedAt = parsedTime

		if updatedAt.Valid {
			g.UpdatedAt, err = parseTime(updatedAt.String)
			if err != nil {
				r.log.Warn("failed to parse updated_at in list", zap.Error(err))
			}
		}

		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", zap.Error(err))
		return nil, 0, err
	}
	log.Info("groups list retrieved successfully",
		zap.Int("returned_count", len(groups)),
		zap.Int64("total_count", total))
	return groups, total, nil
}

// AddSampleToGroup обновляет группу у пробы
func (r *ExperimentGroupRepo) AddSampleToGroup(ctx context.Context, sampleID, groupID string) error {
	log := logQuery(ctx, r.log, "UPDATE", "samples",
		zap.String("sample_id", sampleID),
		zap.String("group_id", groupID))
	log.Debug("linking sample to group")

	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM experiment_groups WHERE id = ?`, groupID).Scan(&exists)
	if err == sql.ErrNoRows {
		log.Warn("group not found for sample linking",
			zap.String("group_id", groupID))
		return fmt.Errorf("group not found")
	}
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx,
		`UPDATE samples SET group_id = ? WHERE id = ?`,
		groupID, sampleID,
	)
	if err != nil {
		return err
	}

	log.Info("sample successfully linked to group")
	return nil
}

// RemoveSampleFromGroup удаляет привязку пробы к группе (устанавливает group_id в NULL)
func (r *ExperimentGroupRepo) RemoveSampleFromGroup(ctx context.Context, sampleID string) error {
	log := logQuery(ctx, r.log, "UPDATE", "samples",
		zap.String("sample_id", sampleID))
	log.Debug("unlinking sample from group")

	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM samples WHERE id = ?`, sampleID).Scan(&exists)
	if err == sql.ErrNoRows {
		log.Warn("sample not found for unlinking",
			zap.String("sample_id", sampleID))
		return fmt.Errorf("sample not found")
	}
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx,
		`UPDATE samples SET group_id = NULL WHERE id = ?`,
		sampleID,
	)
	if err != nil {
		return err
	}

	log.Info("sample successfully unlinked from group")
	return nil
}

func (r *ExperimentGroupRepo) UpdateGroup(ctx context.Context, id string, g models.UpdateExperimentGroup) error {
	log := logQuery(ctx, r.log, "UPDATE", "experiment_groups",
		zap.String("group_id", id),
		zap.Bool("name_provided", g.Name != nil),
		zap.Bool("project_name_provided", g.ProjectName != nil),
		zap.Bool("object_type_provided", g.ObjectType != nil),
		zap.Bool("customer_provided", g.Customer != nil),
		zap.Bool("contract_number_provided", g.ContractNumber != nil),
		zap.Bool("status_provided", g.Status != nil),
		zap.Bool("responsible_person_id_provided", g.ResponsiblePersonID != nil),
		zap.Bool("location_provided", g.Location != nil),
		zap.Bool("description_provided", g.Description != nil))
	log.Debug("preparing dynamic update for group")

	var (
		groupUpdateFields []string
		groupUpdateValues []interface{}
	)
	if g.Name != nil {
		groupUpdateFields = append(groupUpdateFields, "name = ?")
		groupUpdateValues = append(groupUpdateValues, *g.Name)
		log.Debug("including field update", zap.String("field", "name"), zap.String("new_value", *g.Name))
	}
	if g.ProjectName != nil {
		groupUpdateFields = append(groupUpdateFields, "project_name = ?")
		groupUpdateValues = append(groupUpdateValues, *g.ProjectName)
		log.Debug("including field update", zap.String("field", "project_name"), zap.String("new_value", *g.ProjectName))
	}
	if g.ObjectType != nil {
		groupUpdateFields = append(groupUpdateFields, "object_type = ?")
		groupUpdateValues = append(groupUpdateValues, *g.ObjectType)
		log.Debug("including field update", zap.String("field", "object_type"), zap.String("new_value", *g.ObjectType))
	}
	if g.Customer != nil {
		groupUpdateFields = append(groupUpdateFields, "customer = ?")
		groupUpdateValues = append(groupUpdateValues, *g.Customer)
		log.Debug("including field update", zap.String("field", "customer"), zap.String("new_value", *g.Customer))
	}
	if g.ContractNumber != nil {
		groupUpdateFields = append(groupUpdateFields, "contract_number = ?")
		groupUpdateValues = append(groupUpdateValues, *g.ContractNumber)
		log.Debug("including field update", zap.String("field", "contract_number"), zap.String("new_value", *g.ContractNumber))
	}
	if g.Status != nil {
		groupUpdateFields = append(groupUpdateFields, "status = ?")
		groupUpdateValues = append(groupUpdateValues, *g.Status)
		log.Debug("including field update", zap.String("field", "status"), zap.String("new_value", *g.Status))
	}
	if g.ResponsiblePersonID != nil {
		groupUpdateFields = append(groupUpdateFields, "responsible_person_id = ?")
		groupUpdateValues = append(groupUpdateValues, *g.ResponsiblePersonID)
		log.Debug("including field update", zap.String("field", "responsible_person_id"), zap.String("new_value", *g.ResponsiblePersonID))
	}
	if g.Location != nil {
		groupUpdateFields = append(groupUpdateFields, "location = ?")
		groupUpdateValues = append(groupUpdateValues, *g.Location)
		log.Debug("including field update", zap.String("field", "location"), zap.String("new_value", *g.Location))
	}
	if g.Description != nil {
		groupUpdateFields = append(groupUpdateFields, "description = ?")
		groupUpdateValues = append(groupUpdateValues, *g.Description)
		log.Debug("including field update", zap.String("field", "description"), zap.String("new_value", *g.Description))
	}

	if len(groupUpdateFields) == 0 {
		log.Debug("no fields to update, skipping query")
		return nil
	}

	groupUpdateFields = append(groupUpdateFields, "updated_at = ?")
	groupUpdateValues = append(groupUpdateValues, time.Now().Format(timeLayout))

	query := fmt.Sprintf(`UPDATE experiment_groups SET %v WHERE id = ?`, strings.Join(groupUpdateFields, ", "))
	_, err := r.db.ExecContext(ctx, query, append(groupUpdateValues, id)...)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	log.Info("group updated successfully",
		zap.Int("fields_updated", len(groupUpdateFields)))
	return nil
}

func (r *ExperimentGroupRepo) DeleteGroup(ctx context.Context, id string) error {
	log := logQuery(ctx, r.log, "DELETE", "experiment_groups",
		zap.String("group_id", id))
	log.Info("deleting experiment group (transaction)")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Error("transaction rollback failed", zap.Error(rbErr))
			} else {
				log.Debug("transaction rolled back")
			}
		}
	}()

	_, err = tx.ExecContext(ctx, "DELETE FROM experiment_groups WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	if err = tx.Commit(); err != nil {
		log.Error("transaction commit failed", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("experiment group deleted successfully")
	return nil
}
