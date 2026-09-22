package mysql_repo

import (
	"context"
	"database/sql"
	"desktop_lab/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetUserByID получает пользователя по ID (реализация для OrderRepo)
func (r *OrderRepo) GetUserByID(ctx context.Context, id string) (models.User, error) {
	log := r.log.With(zap.String("method", "GetUserByID"), zap.String("user_id", id))
	log.Debug("fetching user by ID")

	var user models.User
	err := r.db.QueryRowContext(ctx, "SELECT id, name, email, role FROM users WHERE id = ? AND deleted_at IS NULL", id).Scan(
		&user.ID, &user.Name, &user.Email, &user.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, fmt.Errorf("user not found")
		}
		return models.User{}, err
	}

	log.Debug("user fetched successfully")
	return user, nil
}

type OrderRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewOrderRepo(db *sqlx.DB, log *zap.Logger) *OrderRepo {
	return &OrderRepo{db: db, log: log}
}

// Create создает новую заявку
func (r *OrderRepo) Create(ctx context.Context, order models.Order) error {
	log := logQuery(ctx, r.log, "INSERT", "orders", zap.String("id", order.ID))
	log.Info("creating new order")

	query := `INSERT INTO orders (
		id, created_by, assigned_to, status, priority, title, description,
		internal_comment, external_comment, total_amount, currency,
		client_name, client_email, client_phone, sample_location, due_date,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var dueDate *string
	if order.DueDate != nil {
		d := order.DueDate.Format(time.RFC3339)
		dueDate = &d
	}

	_, err := r.db.ExecContext(ctx, query,
		order.ID,
		order.CreatedBy,
		order.AssignedTo,
		order.Status,
		order.Priority,
		order.Title,
		order.Description,
		order.InternalComment,
		order.ExternalComment,
		order.TotalAmount,
		order.Currency,
		order.ClientName,
		order.ClientEmail,
		order.ClientPhone,
		order.SampleLocation,
		dueDate,
		order.CreatedAt.Format(time.RFC3339),
		order.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		log.Error("failed to create order", zap.Error(err))
		return err
	}

	log.Info("order created successfully")
	return nil
}

// CreateItem создает позицию заявки
func (r *OrderRepo) CreateItem(ctx context.Context, item models.OrderItem) error {
	log := logQuery(ctx, r.log, "INSERT", "order_items", zap.String("id", item.ID), zap.String("order_id", item.OrderID))
	log.Info("creating order item")

	query := `INSERT INTO order_items (
		id, order_id, test_method_id, test_method_name, quantity, unit_price, subtotal,
		status, sample_required, sample_notes, sample_count, sample_delivered,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	sampleReq := 0
	if item.SampleRequired {
		sampleReq = 1
	}
	sampleDel := 0
	if item.SampleDelivered {
		sampleDel = 1
	}

	_, err := r.db.ExecContext(ctx, query,
		item.ID,
		item.OrderID,
		item.TestMethodID,
		item.TestMethodName,
		item.Quantity,
		item.UnitPrice,
		item.Subtotal,
		item.Status,
		sampleReq,
		item.SampleNotes,
		item.SampleCount,
		sampleDel,
		item.CreatedAt.Format(time.RFC3339),
		item.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		log.Error("failed to create order item", zap.Error(err))
		return err
	}

	log.Info("order item created successfully")
	return nil
}

// GetByID получает заявку по ID
func (r *OrderRepo) GetByID(ctx context.Context, id string) (models.Order, error) {
	log := logQuery(ctx, r.log, "SELECT", "orders", zap.String("id", id))
	log.Debug("fetching order by ID")

	var order models.Order
	var createdAt, updatedAt string
	var completedAt, dueDate sql.NullString
	var assignedTo sql.NullString

	query := `SELECT id, created_by, assigned_to, status, priority, title, description,
		internal_comment, external_comment, total_amount, currency,
		client_name, client_email, client_phone, sample_location, due_date,
		created_at, updated_at, completed_at
		FROM orders WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&order.ID, &order.CreatedBy, &assignedTo, &order.Status,
		&order.Priority, &order.Title, &order.Description,
		&order.InternalComment, &order.ExternalComment,
		&order.TotalAmount, &order.Currency,
		&order.ClientName, &order.ClientEmail, &order.ClientPhone,
		&order.SampleLocation, &dueDate,
		&createdAt, &updatedAt, &completedAt,
	)
	if err != nil {
		log.Error("failed to fetch order", zap.Error(err))
		return models.Order{}, err
	}

	if assignedTo.Valid {
		order.AssignedTo = &assignedTo.String
	}
	order.CreatedAt, _ = parseTime(createdAt)
	order.UpdatedAt, _ = parseTime(updatedAt)
	if completedAt.Valid {
		t, _ := parseTime(completedAt.String)
		order.CompletedAt = &t
	}
	if dueDate.Valid {
		t, _ := parseTime(dueDate.String)
		order.DueDate = &t
	}

	log.Debug("order retrieved successfully")
	return order, nil
}

// GetWithItemsAndWorkflow получает заявку с позициями и историей workflow
func (r *OrderRepo) GetWithItemsAndWorkflow(ctx context.Context, id string) (models.OrderResponse, error) {
	log := logQuery(ctx, r.log, "SELECT", "orders with items and workflow", zap.String("id", id))
	log.Debug("fetching order full details")

	response := models.OrderResponse{}

	order, err := r.GetByID(ctx, id)
	if err != nil {
		return response, err
	}
	response.Order = order

	items, err := r.GetItemsByOrderID(ctx, id)
	if err != nil {
		return response, err
	}
	response.Items = items

	workflow, err := r.GetWorkflowByOrderID(ctx, id)
	if err != nil {
		return response, err
	}
	response.Workflow = workflow

	log.Debug("order full details retrieved successfully")
	return response, nil
}

// GetItemsByOrderID получает все позиции заявки
func (r *OrderRepo) GetItemsByOrderID(ctx context.Context, orderID string) ([]models.OrderItem, error) {
	log := logQuery(ctx, r.log, "SELECT", "order_items", zap.String("order_id", orderID))
	log.Debug("fetching order items")

	query := `SELECT id, order_id, test_method_id, test_method_name, quantity, unit_price, subtotal,
		status, sample_required, sample_notes, sample_count, sample_delivered,
		created_at, updated_at
		FROM order_items WHERE order_id = ? ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, orderID)
	if err != nil {
		log.Error("failed to fetch order items", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		var createdAt, updatedAt string
		var sampleRequired, sampleDelivered int

		err := rows.Scan(
			&item.ID, &item.OrderID, &item.TestMethodID, &item.TestMethodName,
			&item.Quantity, &item.UnitPrice, &item.Subtotal, &item.Status,
			&sampleRequired, &item.SampleNotes, &item.SampleCount, &sampleDelivered,
			&createdAt, &updatedAt,
		)
		if err != nil {
			log.Error("failed to scan order item", zap.Error(err))
			return nil, err
		}

		item.SampleRequired = sampleRequired == 1
		item.SampleDelivered = sampleDelivered == 1
		item.CreatedAt, _ = parseTime(createdAt)
		item.UpdatedAt, _ = parseTime(updatedAt)

		items = append(items, item)
	}

	log.Debug("order items retrieved", zap.Int("count", len(items)))
	return items, nil
}

// GetWorkflowByOrderID получает историю изменения статусов заявки
func (r *OrderRepo) GetWorkflowByOrderID(ctx context.Context, orderID string) ([]models.OrderWorkflowEntry, error) {
	log := logQuery(ctx, r.log, "SELECT", "order_workflow", zap.String("order_id", orderID))
	log.Debug("fetching order workflow")

	query := `SELECT id, order_id, from_status, to_status, user_id, user_name, comment, created_at
		FROM order_workflow WHERE order_id = ? ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, orderID)
	if err != nil {
		log.Error("failed to fetch order workflow", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var workflow []models.OrderWorkflowEntry
	for rows.Next() {
		var entry models.OrderWorkflowEntry
		var createdAt string

		err := rows.Scan(
			&entry.ID, &entry.OrderID, &entry.FromStatus, &entry.ToStatus,
			&entry.UserID, &entry.UserName, &entry.Comment, &createdAt,
		)
		if err != nil {
			log.Error("failed to scan workflow entry", zap.Error(err))
			return nil, err
		}

		entry.CreatedAt, _ = parseTime(createdAt)
		workflow = append(workflow, entry)
	}

	log.Debug("order workflow retrieved", zap.Int("count", len(workflow)))
	return workflow, nil
}

// AddWorkflowEntry добавляет запись в историю workflow
func (r *OrderRepo) AddWorkflowEntry(ctx context.Context, entry models.OrderWorkflowEntry) error {
	log := logQuery(ctx, r.log, "INSERT", "order_workflow", zap.String("order_id", entry.OrderID))
	log.Info("adding workflow entry")

	entry.ID = uuid.New().String()

	query := `INSERT INTO order_workflow (id, order_id, from_status, to_status, user_id, user_name, comment, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID, entry.OrderID, entry.FromStatus, entry.ToStatus,
		entry.UserID, entry.UserName, entry.Comment, time.Now().Format(time.RFC3339),
	)
	if err != nil {
		log.Error("failed to add workflow entry", zap.Error(err))
		return err
	}

	log.Info("workflow entry added successfully")
	return nil
}

// List получает список заявок с фильтрацией и пагинацией
func (r *OrderRepo) List(ctx context.Context, filter models.OrderListFilter) ([]models.Order, int64, error) {
	log := logQuery(ctx, r.log, "SELECT", "orders", zap.Any("filter", filter))
	log.Debug("fetching orders list")

	baseQuery := `FROM orders WHERE 1=1`
	countQuery := `SELECT COUNT(*) ` + baseQuery
	args := []interface{}{}

	if filter.CreatedBy != "" {
		baseQuery += ` AND created_by = ?`
		args = append(args, filter.CreatedBy)
	}
	if filter.AssignedTo != "" {
		baseQuery += ` AND assigned_to = ?`
		args = append(args, filter.AssignedTo)
	}
	if filter.Status != "" {
		baseQuery += ` AND status = ?`
		args = append(args, filter.Status)
	}
	if filter.Priority != "" {
		baseQuery += ` AND priority = ?`
		args = append(args, filter.Priority)
	}
	if filter.DateFrom != "" {
		baseQuery += ` AND created_at >= ?`
		args = append(args, filter.DateFrom)
	}
	if filter.DateTo != "" {
		baseQuery += ` AND created_at <= ?`
		args = append(args, filter.DateTo)
	}
	if filter.SearchQuery != "" {
		baseQuery += ` AND (title LIKE ? OR description LIKE ? OR client_name LIKE ?)`
		searchPattern := "%" + filter.SearchQuery + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		log.Error("failed to count orders", zap.Error(err))
		return nil, 0, err
	}

	if total == 0 {
		return nil, 0, nil
	}

	selectQuery := `SELECT id, created_by, assigned_to, status, priority, title, description,
		internal_comment, external_comment, total_amount, currency,
		client_name, client_email, client_phone, sample_location, due_date,
		created_at, updated_at, completed_at ` + baseQuery + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		log.Error("failed to fetch orders", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		var createdAt, updatedAt string
		var completedAt, dueDate sql.NullString
		var assignedTo sql.NullString

		err := rows.Scan(
			&order.ID, &order.CreatedBy, &assignedTo, &order.Status,
			&order.Priority, &order.Title, &order.Description,
			&order.InternalComment, &order.ExternalComment,
			&order.TotalAmount, &order.Currency,
			&order.ClientName, &order.ClientEmail, &order.ClientPhone,
			&order.SampleLocation, &dueDate,
			&createdAt, &updatedAt, &completedAt,
		)
		if err != nil {
			log.Error("failed to scan order", zap.Error(err))
			return nil, 0, err
		}

		if assignedTo.Valid {
			order.AssignedTo = &assignedTo.String
		}
		order.CreatedAt, _ = parseTime(createdAt)
		order.UpdatedAt, _ = parseTime(updatedAt)
		if completedAt.Valid {
			t, _ := parseTime(completedAt.String)
			order.CompletedAt = &t
		}
		if dueDate.Valid {
			t, _ := parseTime(dueDate.String)
			order.DueDate = &t
		}

		orders = append(orders, order)
	}

	log.Debug("orders list retrieved", zap.Int64("total", total), zap.Int("count", len(orders)))
	return orders, total, nil
}

// Update обновляет заявку
func (r *OrderRepo) Update(ctx context.Context, id string, req models.UpdateOrderRequest) (models.Order, error) {
	log := logQuery(ctx, r.log, "UPDATE", "orders", zap.String("id", id))
	log.Debug("updating order")

	setClauses := []string{"updated_at = ?"}
	args := []interface{}{time.Now().Format(time.RFC3339)}

	if req.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *req.Title)
	}
	if req.Description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *req.Description)
	}
	if req.Priority != nil {
		setClauses = append(setClauses, "priority = ?")
		args = append(args, *req.Priority)
	}
	if req.ClientName != nil {
		setClauses = append(setClauses, "client_name = ?")
		args = append(args, *req.ClientName)
	}
	if req.ClientEmail != nil {
		setClauses = append(setClauses, "client_email = ?")
		args = append(args, *req.ClientEmail)
	}
	if req.ClientPhone != nil {
		setClauses = append(setClauses, "client_phone = ?")
		args = append(args, *req.ClientPhone)
	}
	if req.ExternalComment != nil {
		setClauses = append(setClauses, "external_comment = ?")
		args = append(args, *req.ExternalComment)
	}
	if req.InternalComment != nil {
		setClauses = append(setClauses, "internal_comment = ?")
		args = append(args, *req.InternalComment)
	}
	if req.SampleLocation != nil {
		setClauses = append(setClauses, "sample_location = ?")
		args = append(args, *req.SampleLocation)
	}
	if req.DueDate != nil {
		setClauses = append(setClauses, "due_date = ?")
		if req.DueDate.IsZero() {
			args = append(args, (*string)(nil))
		} else {
			args = append(args, req.DueDate.Format(time.RFC3339))
		}
	}

	args = append(args, id)
	query := fmt.Sprintf(`UPDATE orders SET %s WHERE id = ?`, joinStrings(setClauses, ", "))

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error("update failed", zap.Error(err))
		return models.Order{}, err
	}

	log.Debug("order updated")
	return r.GetByID(ctx, id)
}

// UpdateStatus обновляет статус заявки и добавляет запись в workflow
func (r *OrderRepo) UpdateStatus(ctx context.Context, id, newStatus, userID, userName, comment string) error {
	log := logQuery(ctx, r.log, "UPDATE", "orders", zap.String("id", id), zap.String("status", newStatus))
	log.Info("updating order status")

	order, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	query := `UPDATE orders SET status = ?, updated_at = ?`
	args := []interface{}{newStatus, time.Now().Format(time.RFC3339)}

	if newStatus == string(models.OrderStatusCompleted) || newStatus == string(models.OrderStatusCancelled) {
		query += `, completed_at = ?`
		completedTime := time.Now().Format(time.RFC3339)
		args = append(args, completedTime)
	}

	args = append(args, id)
	query += ` WHERE id = ?`

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error("status update failed", zap.Error(err))
		return err
	}

	entry := models.OrderWorkflowEntry{
		OrderID:    id,
		FromStatus: string(order.Status),
		ToStatus:   newStatus,
		UserID:     userID,
		UserName:   userName,
		Comment:    comment,
	}

	if err := r.AddWorkflowEntry(ctx, entry); err != nil {
		log.Warn("failed to add workflow entry", zap.Error(err))
	}

	log.Info("order status updated successfully")
	return nil
}

// UpdateItem обновляет позицию заявки
func (r *OrderRepo) UpdateItem(ctx context.Context, id string, req models.UpdateOrderItemRequest) (models.OrderItem, error) {
	log := logQuery(ctx, r.log, "UPDATE", "order_items", zap.String("id", id))
	log.Debug("updating order item")

	setClauses := []string{"updated_at = ?"}
	args := []interface{}{time.Now().Format(time.RFC3339)}

	if req.Quantity != nil {
		setClauses = append(setClauses, "quantity = ?")
		args = append(args, *req.Quantity)
	}
	if req.SampleNotes != nil {
		setClauses = append(setClauses, "sample_notes = ?")
		args = append(args, *req.SampleNotes)
	}
	if req.SampleCount != nil {
		setClauses = append(setClauses, "sample_count = ?")
		args = append(args, *req.SampleCount)
	}
	if req.SampleDelivered != nil {
		setClauses = append(setClauses, "sample_delivered = ?")
		args = append(args, boolToInt(*req.SampleDelivered))
	}

	args = append(args, id)
	query := fmt.Sprintf(`UPDATE order_items SET %s WHERE id = ?`, joinStrings(setClauses, ", "))

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error("item update failed", zap.Error(err))
		return models.OrderItem{}, err
	}

	log.Debug("order item updated")
	return r.GetItemByID(ctx, id)
}

// GetItemByID получает позицию заявки по ID
func (r *OrderRepo) GetItemByID(ctx context.Context, id string) (models.OrderItem, error) {
	log := logQuery(ctx, r.log, "SELECT", "order_items", zap.String("id", id))
	log.Debug("fetching order item by ID")

	var item models.OrderItem
	var createdAt, updatedAt string
	var sampleRequired, sampleDelivered int

	query := `SELECT id, order_id, test_method_id, test_method_name, quantity, unit_price, subtotal,
		status, sample_required, sample_notes, sample_count, sample_delivered,
		created_at, updated_at
		FROM order_items WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&item.ID, &item.OrderID, &item.TestMethodID, &item.TestMethodName,
		&item.Quantity, &item.UnitPrice, &item.Subtotal, &item.Status,
		&sampleRequired, &item.SampleNotes, &item.SampleCount, &sampleDelivered,
		&createdAt, &updatedAt,
	)
	if err != nil {
		log.Error("failed to fetch order item", zap.Error(err))
		return models.OrderItem{}, err
	}

	item.SampleRequired = sampleRequired == 1
	item.SampleDelivered = sampleDelivered == 1
	item.CreatedAt, _ = parseTime(createdAt)
	item.UpdatedAt, _ = parseTime(updatedAt)

	log.Debug("order item retrieved successfully")
	return item, nil
}

// Delete удаляет заявку (мягкое удаление через статус cancelled)
func (r *OrderRepo) Delete(ctx context.Context, id string) error {
	log := logQuery(ctx, r.log, "UPDATE", "orders", zap.String("id", id))
	log.Info("soft deleting order")

	query := `UPDATE orders SET status = 'cancelled', updated_at = ?, completed_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339), id)
	if err != nil {
		log.Error("soft delete failed", zap.Error(err))
		return err
	}

	log.Info("order soft deleted successfully")
	return nil
}

// DeleteItem удаляет позицию заявки
func (r *OrderRepo) DeleteItem(ctx context.Context, id string) error {
	log := logQuery(ctx, r.log, "DELETE", "order_items", zap.String("id", id))
	log.Debug("deleting order item")

	query := `DELETE FROM order_items WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		log.Error("delete failed", zap.Error(err))
		return err
	}

	log.Info("order item deleted successfully")
	return nil
}

// RecalculateTotal пересчитывает общую сумму заявки
func (r *OrderRepo) RecalculateTotal(ctx context.Context, orderID string) error {
	log := logQuery(ctx, r.log, "UPDATE", "orders recalculate", zap.String("order_id", orderID))
	log.Debug("recalculating order total")

	query := `UPDATE orders SET total_amount = (
		SELECT COALESCE(SUM(subtotal), 0) FROM order_items WHERE order_id = ?
	), updated_at = ? WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, orderID, time.Now().Format(time.RFC3339), orderID)
	if err != nil {
		log.Error("recalculation failed", zap.Error(err))
		return err
	}

	log.Info("order total recalculated successfully")
	return nil
}

// AssignOrder назначает ответственного за заявку
func (r *OrderRepo) AssignOrder(ctx context.Context, id, assignedTo, userID, userName, comment string) error {
	log := logQuery(ctx, r.log, "UPDATE", "orders", zap.String("id", id), zap.String("assigned_to", assignedTo))
	log.Info("assigning order to manager")

	order, err := r.GetByID(ctx, id)
	if err != nil {
		log.Error("failed to get order", zap.Error(err))
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return err
	}
	defer tx.Rollback()

	// Обновляем ответственного
	updateQuery := `UPDATE orders SET assigned_to = ?, status = ?, updated_at = ? WHERE id = ?`
	_, err = tx.ExecContext(ctx, updateQuery, assignedTo, models.OrderStatusAssigned, time.Now().Format(time.RFC3339), id)
	if err != nil {
		log.Error("failed to update order", zap.Error(err))
		return err
	}

	// Добавляем запись в workflow
	entry := models.OrderWorkflowEntry{
		ID:         uuid.New().String(),
		OrderID:    id,
		FromStatus: string(order.Status),
		ToStatus:   string(models.OrderStatusAssigned),
		UserID:     userID,
		UserName:   userName,
		Comment:    comment,
		CreatedAt:  time.Now(),
	}

	insertQuery := `INSERT INTO order_workflow (id, order_id, from_status, to_status, user_id, user_name, comment, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, insertQuery, entry.ID, entry.OrderID, entry.FromStatus, entry.ToStatus, entry.UserID, entry.UserName, entry.Comment, entry.CreatedAt.Format(time.RFC3339))
	if err != nil {
		log.Error("failed to add workflow entry", zap.Error(err))
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error("failed to commit transaction", zap.Error(err))
		return err
	}

	log.Info("order assigned successfully")
	return nil
}

// AssignOrderItem назначает исполнителя для позиции заявки
func (r *OrderRepo) AssignOrderItem(ctx context.Context, id, assignedTo, userID, userName, comment string) error {
	log := logQuery(ctx, r.log, "UPDATE", "order_items", zap.String("id", id), zap.String("assigned_to", assignedTo))
	log.Info("assigning order item to executor")

	item, err := r.GetItemByID(ctx, id)
	if err != nil {
		log.Error("failed to get order item", zap.Error(err))
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return err
	}
	defer tx.Rollback()

	// Обновляем исполнителя и статус
	updateQuery := `UPDATE order_items SET assigned_to = ?, status = ?, updated_at = ? WHERE id = ?`
	_, err = tx.ExecContext(ctx, updateQuery, assignedTo, models.OrderItemStatusAssigned, time.Now().Format(time.RFC3339), id)
	if err != nil {
		log.Error("failed to update order item", zap.Error(err))
		return err
	}

	// Добавляем запись в workflow родительской заявки
	entry := models.OrderWorkflowEntry{
		ID:         uuid.New().String(),
		OrderID:    item.OrderID,
		FromStatus: string(models.OrderStatusAssigned),
		ToStatus:   string(models.OrderStatusInProgress),
		UserID:     userID,
		UserName:   userName,
		Comment:    fmt.Sprintf("Назначен исполнитель на позицию: %s (%s)", comment, id),
		CreatedAt:  time.Now(),
	}

	insertQuery := `INSERT INTO order_workflow (id, order_id, from_status, to_status, user_id, user_name, comment, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, insertQuery, entry.ID, entry.OrderID, entry.FromStatus, entry.ToStatus, entry.UserID, entry.UserName, entry.Comment, entry.CreatedAt.Format(time.RFC3339))
	if err != nil {
		log.Error("failed to add workflow entry", zap.Error(err))
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error("failed to commit transaction", zap.Error(err))
		return err
	}

	log.Info("order item assigned successfully")
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
