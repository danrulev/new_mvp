package mysql_repo

import (
"context"
"desktop_lab/internal/models"
"fmt"
"time"

"database/sql"
"github.com/jmoiron/sqlx"
"go.uber.org/zap"
)

type InvitationRepo struct {
db  *sqlx.DB
log *zap.Logger
}

func NewInvitationRepo(db *sqlx.DB, log *zap.Logger) *InvitationRepo {
return &InvitationRepo{db: db, log: log}
}

// Create создает новое приглашение
func (r *InvitationRepo) Create(ctx context.Context, invitation models.OrganizationInvitation) error {
logger := logQuery(ctx, r.log, "INSERT", "organization_invitations", zap.String("id", invitation.ID))
logger.Info("creating new invitation")

query := `INSERT INTO organization_invitations (
id, organization_id, email, role, invited_by, status, token, expires_at, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

_, err := r.db.ExecContext(ctx, query,
invitation.ID,
invitation.OrganizationID,
invitation.Email,
invitation.Role,
invitation.InvitedBy,
invitation.Status,
invitation.Token,
invitation.ExpiresAt.Format(time.RFC3339),
invitation.CreatedAt.Format(time.RFC3339),
)
if err != nil {
logger.Error("failed to create invitation", zap.Error(err))
return err
}

logger.Info("invitation created successfully")
return nil
}

// GetByID получает приглашение по ID
func (r *InvitationRepo) GetByID(ctx context.Context, id string) (models.OrganizationInvitation, error) {
logger := logQuery(ctx, r.log, "SELECT", "organization_invitations", zap.String("id", id))
logger.Debug("fetching invitation by ID")

var invitation models.OrganizationInvitation
var createdAt, expiresAt, acceptedAt sql.NullString

query := `SELECT id, organization_id, email, role, invited_by, status, token, expires_at, created_at, accepted_at
FROM organization_invitations WHERE id = ?`

row := r.db.QueryRowContext(ctx, query, id)
err := row.Scan(
&invitation.ID, &invitation.OrganizationID, &invitation.Email, &invitation.Role,
&invitation.InvitedBy, &invitation.Status, &invitation.Token,
&expiresAt, &createdAt, &acceptedAt,
)
if err != nil {
logger.Error("failed to fetch invitation", zap.Error(err))
return models.OrganizationInvitation{}, err
}

invitation.CreatedAt, _ = parseTime(createdAt.String)
invitation.ExpiresAt, _ = parseTime(expiresAt.String)
if acceptedAt.Valid && acceptedAt.String != "" {
t, _ := parseTime(acceptedAt.String)
invitation.AcceptedAt = &t
}

logger.Debug("invitation retrieved successfully")
return invitation, nil
}

// GetByToken получает приглашение по токену
func (r *InvitationRepo) GetByToken(ctx context.Context, token string) (models.OrganizationInvitation, error) {
logger := logQuery(ctx, r.log, "SELECT", "organization_invitations", zap.String("token", token[:8]+"..."))
logger.Debug("fetching invitation by token")

var invitation models.OrganizationInvitation
var createdAt, expiresAt, acceptedAt sql.NullString

query := `SELECT id, organization_id, email, role, invited_by, status, token, expires_at, created_at, accepted_at
FROM organization_invitations WHERE token = ?`

row := r.db.QueryRowContext(ctx, query, token)
err := row.Scan(
&invitation.ID, &invitation.OrganizationID, &invitation.Email, &invitation.Role,
&invitation.InvitedBy, &invitation.Status, &invitation.Token,
&expiresAt, &createdAt, &acceptedAt,
)
if err != nil {
logger.Error("failed to fetch invitation by token", zap.Error(err))
return models.OrganizationInvitation{}, err
}

invitation.CreatedAt, _ = parseTime(createdAt.String)
invitation.ExpiresAt, _ = parseTime(expiresAt.String)
if acceptedAt.Valid && acceptedAt.String != "" {
t, _ := parseTime(acceptedAt.String)
invitation.AcceptedAt = &t
}

logger.Debug("invitation retrieved successfully by token")
return invitation, nil
}

// List получает список приглашений с фильтрацией
func (r *InvitationRepo) List(ctx context.Context, filter models.InvitationListFilter) ([]models.OrganizationInvitation, int64, error) {
logger := logQuery(ctx, r.log, "SELECT", "organization_invitations", zap.Any("filter", filter))
logger.Debug("fetching invitations list")

baseQuery := `FROM organization_invitations WHERE 1=1`
countQuery := `SELECT COUNT(*) ` + baseQuery
args := []interface{}{}

if filter.OrganizationID != "" {
baseQuery += ` AND organization_id = ?`
args = append(args, filter.OrganizationID)
}
if filter.Email != "" {
baseQuery += ` AND email LIKE ?`
args = append(args, "%"+filter.Email+"%")
}
if filter.Status != "" {
baseQuery += ` AND status = ?`
args = append(args, filter.Status)
}
if filter.InvitedBy != "" {
baseQuery += ` AND invited_by = ?`
args = append(args, filter.InvitedBy)
}

var total int64
err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
if err != nil {
logger.Error("failed to count invitations", zap.Error(err))
return nil, 0, err
}

if total == 0 {
return nil, 0, nil
}

selectQuery := `SELECT id, organization_id, email, role, invited_by, status, token, expires_at, created_at, accepted_at ` + 
baseQuery + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`

args = append(args, filter.Limit, filter.Offset)

rows, err := r.db.QueryContext(ctx, selectQuery, args...)
if err != nil {
logger.Error("failed to fetch invitations", zap.Error(err))
return nil, 0, err
}
defer rows.Close()

var invitations []models.OrganizationInvitation
for rows.Next() {
var inv models.OrganizationInvitation
var createdAt, expiresAt, acceptedAt sql.NullString
var token string

err := rows.Scan(
&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role,
&inv.InvitedBy, &inv.Status, &token,
&expiresAt, &createdAt, &acceptedAt,
)
if err != nil {
logger.Error("failed to scan invitation", zap.Error(err))
return nil, 0, err
}

inv.CreatedAt, _ = parseTime(createdAt.String)
inv.ExpiresAt, _ = parseTime(expiresAt.String)
if acceptedAt.Valid && acceptedAt.String != "" {
t, _ := parseTime(acceptedAt.String)
inv.AcceptedAt = &t
}
// Не возвращаем токен в списке
inv.Token = ""

invitations = append(invitations, inv)
}

logger.Debug("invitations list retrieved", zap.Int64("total", total), zap.Int("count", len(invitations)))
return invitations, total, nil
}

// UpdateStatus обновляет статус приглашения
func (r *InvitationRepo) UpdateStatus(ctx context.Context, id string, status models.InvitationStatus, acceptedAt *time.Time) error {
logger := logQuery(ctx, r.log, "UPDATE", "organization_invitations", zap.String("id", id), zap.String("status", string(status)))
logger.Info("updating invitation status")

var query string
var args []interface{}

if acceptedAt != nil {
query = `UPDATE organization_invitations SET status = ?, accepted_at = ? WHERE id = ?`
args = []interface{}{status, acceptedAt.Format(time.RFC3339), id}
} else {
query = `UPDATE organization_invitations SET status = ? WHERE id = ?`
args = []interface{}{status, id}
}

_, err := r.db.ExecContext(ctx, query, args...)
if err != nil {
logger.Error("failed to update invitation status", zap.Error(err))
return err
}

logger.Info("invitation status updated successfully")
return nil
}

// Delete удаляет приглашение
func (r *InvitationRepo) Delete(ctx context.Context, id string) error {
logger := logQuery(ctx, r.log, "DELETE", "organization_invitations", zap.String("id", id))
logger.Info("deleting invitation")

query := `DELETE FROM organization_invitations WHERE id = ?`
_, err := r.db.ExecContext(ctx, query, id)
if err != nil {
logger.Error("failed to delete invitation", zap.Error(err))
return err
}

logger.Info("invitation deleted successfully")
return nil
}

// GetPendingByOrgAndEmail получает активные приглашения для организации и email
func (r *InvitationRepo) GetPendingByOrgAndEmail(ctx context.Context, orgID, email string) ([]models.OrganizationInvitation, error) {
logger := logQuery(ctx, r.log, "SELECT", "organization_invitations", zap.String("org_id", orgID), zap.String("email", email))
logger.Debug("fetching pending invitations")

query := `SELECT id, organization_id, email, role, invited_by, status, token, expires_at, created_at, accepted_at
FROM organization_invitations 
WHERE organization_id = ? AND email = ? AND status = 'pending'
ORDER BY created_at DESC`

rows, err := r.db.QueryContext(ctx, query, orgID, email)
if err != nil {
logger.Error("failed to fetch pending invitations", zap.Error(err))
return nil, err
}
defer rows.Close()

var invitations []models.OrganizationInvitation
for rows.Next() {
var inv models.OrganizationInvitation
var createdAt, expiresAt, acceptedAt sql.NullString
var token string

err := rows.Scan(
&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role,
&inv.InvitedBy, &inv.Status, &token,
&expiresAt, &createdAt, &acceptedAt,
)
if err != nil {
logger.Error("failed to scan invitation", zap.Error(err))
return nil, err
}

inv.CreatedAt, _ = parseTime(createdAt.String)
inv.ExpiresAt, _ = parseTime(expiresAt.String)
if acceptedAt.Valid && acceptedAt.String != "" {
t, _ := parseTime(acceptedAt.String)
inv.AcceptedAt = &t
}

invitations = append(invitations, inv)
}

logger.Debug("pending invitations retrieved", zap.Int("count", len(invitations)))
return invitations, nil
}

// UpdateTokenAndExpires обновляет токен и срок действия приглашения
func (r *InvitationRepo) UpdateTokenAndExpires(ctx context.Context, id, token string, expiresAt time.Time) error {
logger := logQuery(ctx, r.log, "UPDATE", "organization_invitations", zap.String("id", id))
logger.Info("updating invitation token and expiry")

query := `UPDATE organization_invitations SET token = ?, expires_at = ?, status = 'pending' WHERE id = ?`
_, err := r.db.ExecContext(ctx, query, token, expiresAt.Format(time.RFC3339), id)
if err != nil {
logger.Error("failed to update invitation token and expiry", zap.Error(err))
return err
}

logger.Info("invitation token and expiry updated successfully")
return nil
}

// AddOrganizationUser добавляет пользователя в организацию
// Этот метод нужен для принятия приглашения
func (r *InvitationRepo) AddOrganizationUser(ctx context.Context, orgUser models.OrganizationUser) error {
logger := logQuery(ctx, r.log, "INSERT", "organization_users", zap.String("user_id", orgUser.UserID), zap.String("org_id", orgUser.OrganizationID))
logger.Info("adding user to organization")

// Проверяем, не состоит ли уже пользователь в организации
checkQuery := `SELECT COUNT(*) FROM organization_users WHERE organization_id = ? AND user_id = ?`
var count int
err := r.db.QueryRowContext(ctx, checkQuery, orgUser.OrganizationID, orgUser.UserID).Scan(&count)
if err != nil {
logger.Error("failed to check existing membership", zap.Error(err))
return err
}

if count > 0 {
logger.Warn("user already member of organization")
return fmt.Errorf("пользователь уже состоит в организации")
}

query := `INSERT INTO organization_users (id, organization_id, user_id, role, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`

_, err = r.db.ExecContext(ctx, query,
orgUser.ID,
orgUser.OrganizationID,
orgUser.UserID,
orgUser.Role,
orgUser.CreatedAt.Format(time.RFC3339),
orgUser.UpdatedAt.Format(time.RFC3339),
)
if err != nil {
logger.Error("failed to add user to organization", zap.Error(err))
return err
}

logger.Info("user added to organization successfully")
return nil
}
