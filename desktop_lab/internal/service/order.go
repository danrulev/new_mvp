package service

import (
	"context"
	"desktop_lab/internal/models"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// OrderRepo интерфейс репозитория для работы с заявками
type OrderRepo interface {
	Create(ctx context.Context, order models.Order) error
	CreateItem(ctx context.Context, item models.OrderItem) error
	GetByID(ctx context.Context, id string) (models.Order, error)
	GetWithItemsAndWorkflow(ctx context.Context, id string) (models.OrderResponse, error)
	GetItemsByOrderID(ctx context.Context, orderID string) ([]models.OrderItem, error)
	GetWorkflowByOrderID(ctx context.Context, orderID string) ([]models.OrderWorkflowEntry, error)
	AddWorkflowEntry(ctx context.Context, entry models.OrderWorkflowEntry) error
	List(ctx context.Context, filter models.OrderListFilter) ([]models.Order, int64, error)
	Update(ctx context.Context, id string, req models.UpdateOrderRequest) (models.Order, error)
	UpdateStatus(ctx context.Context, id, newStatus, userID, userName, comment string) error
	UpdateItem(ctx context.Context, id string, req models.UpdateOrderItemRequest) (models.OrderItem, error)
	GetItemByID(ctx context.Context, id string) (models.OrderItem, error)
	Delete(ctx context.Context, id string) error
	DeleteItem(ctx context.Context, id string) error
	RecalculateTotal(ctx context.Context, orderID string) error
	AssignOrder(ctx context.Context, id, assignedTo, userID, userName, comment string) error
	AssignOrderItem(ctx context.Context, id, assignedTo, userID, userName, comment string) error
	GetUserByID(ctx context.Context, id string) (models.User, error)
}

type OrderService struct {
	repo OrderRepo
	log  *zap.Logger
}

func NewOrderService(repo OrderRepo, log *zap.Logger) *OrderService {
	return &OrderService{
		repo: repo,
		log:  log,
	}
}

// Ошибки сервиса заявок
var (
	ErrOrderNotFound      = errors.New("заявка не найдена")
	ErrInvalidStatus      = errors.New("недопустимый статус")
	ErrInvalidTransition  = errors.New("недопустимый переход статуса")
	ErrUserNotFound       = errors.New("пользователь не найден")
	ErrInvalidPriority    = errors.New("недопустимый приоритет")
)

// CreateOrder создает новую заявку на исследования
func (s *OrderService) CreateOrder(ctx context.Context, req models.CreateOrderRequest, userID string) (models.Order, error) {
	logger := loggerWith(ctx, s.log, zap.String("operation", "CreateOrder"))
	logger.Info("creating new order")

	// Валидация приоритета
	priority := req.Priority
	if priority == "" {
		priority = "normal"
	}
	if !isValidPriority(priority) {
		return models.Order{}, ErrInvalidPriority
	}

	orderID := uuid.New().String()
	now := time.Now()

	order := models.Order{
		ID:              orderID,
		CreatedBy:       userID,
		Status:          models.OrderStatusNew,
		Priority:        priority,
		Title:           req.Title,
		Description:     req.Description,
		InternalComment: req.InternalComment,
		ExternalComment: req.ExternalComment,
		Currency:        "RUB",
		ClientName:      req.ClientName,
		ClientEmail:     req.ClientEmail,
		ClientPhone:     req.ClientPhone,
		SampleLocation:  req.SampleLocation,
		DueDate:         req.DueDate,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx, order); err != nil {
		logger.Error("failed to create order", zap.Error(err))
		return models.Order{}, err
	}

	// Создаем позиции заявки
	for _, itemReq := range req.Items {
		item := models.OrderItem{
			ID:             uuid.New().String(),
			OrderID:        orderID,
			TestMethodID:   itemReq.TestMethodID,
			Quantity:       itemReq.Quantity,
			UnitPrice:      itemReq.UnitPrice,
			SampleRequired: itemReq.SampleRequired,
			SampleNotes:    itemReq.SampleNotes,
			SampleCount:    itemReq.SampleCount,
			Status:         models.OrderItemStatusPending,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		item.Subtotal = float64(item.Quantity) * item.UnitPrice

		if err := s.repo.CreateItem(ctx, item); err != nil {
			logger.Error("failed to create order item", zap.Error(err))
			return models.Order{}, err
		}
	}

	if err := s.repo.RecalculateTotal(ctx, orderID); err != nil {
		logger.Warn("failed to recalculate total", zap.Error(err))
	}

	// Добавляем запись в workflow
	workflowEntry := models.OrderWorkflowEntry{
		OrderID:    orderID,
		FromStatus: "",
		ToStatus:   string(models.OrderStatusNew),
		UserID:     userID,
		UserName:   "", // Будет заполнено при получении пользователя
		Comment:    "Заявка создана",
		CreatedAt:  now,
	}
	if err := s.repo.AddWorkflowEntry(ctx, workflowEntry); err != nil {
		logger.Warn("failed to add workflow entry", zap.Error(err))
	}

	logger.Info("order created successfully", zap.String("order_id", orderID))
	return s.repo.GetByID(ctx, orderID)
}

// isValidPriority проверяет допустимость приоритета
func isValidPriority(priority string) bool {
	validPriorities := map[string]bool{
		"normal": true,
		"high":   true,
		"urgent": true,
	}
	return validPriorities[priority]
}

// GetOrder получает заявку по ID
func (s *OrderService) GetOrder(ctx context.Context, id string) (models.OrderResponse, error) {
	logger := loggerWith(ctx, s.log, zap.String("order_id", id), zap.String("operation", "GetOrder"))
	logger.Debug("fetching order")

	return s.repo.GetWithItemsAndWorkflow(ctx, id)
}

// ListOrders получает список заявок с фильтрацией
func (s *OrderService) ListOrders(ctx context.Context, filter models.OrderListFilter) (models.OrderListResponse, error) {
	logger := loggerWith(ctx, s.log, zap.Any("filter", filter), zap.String("operation", "ListOrders"))
	logger.Debug("fetching orders list")

	orders, total, err := s.repo.List(ctx, filter)
	if err != nil {
		logger.Error("failed to get orders list from repo", zap.Error(err))
		return models.OrderListResponse{}, err
	}

	return models.OrderListResponse{Orders: orders, Meta: models.MakePaginatedMetadata(filter.Limit, filter.Offset, total)}, nil
}

// UpdateOrder обновляет заявку
func (s *OrderService) UpdateOrder(ctx context.Context, id string, req models.UpdateOrderRequest) (models.Order, error) {
	logger := loggerWith(ctx, s.log, zap.String("order_id", id), zap.String("operation", "UpdateOrder"))
	logger.Debug("updating order")

	return s.repo.Update(ctx, id, req)
}

// ChangeOrderStatus изменяет статус заявки
func (s *OrderService) ChangeOrderStatus(ctx context.Context, id, newStatus, userID, userName, comment string) error {
	logger := loggerWith(ctx, s.log,
		zap.String("order_id", id),
		zap.String("new_status", newStatus),
		zap.String("user_id", userID),
		zap.String("operation", "ChangeOrderStatus"))
	logger.Info("changing order status")

	status := models.OrderStatus(newStatus)
	if !status.IsValid() {
		logger.Error("invalid status", zap.String("status", newStatus))
		return fmt.Errorf("недопустимый статус")
	}

	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		logger.Error("failed to get order", zap.Error(err))
		return err
	}

	if err := s.validateStatusTransition(order.Status, status); err != nil {
		logger.Error("invalid status transition", zap.Error(err))
		return err
	}

	if err := s.repo.UpdateStatus(ctx, id, newStatus, userID, userName, comment); err != nil {
		logger.Error("failed to update status", zap.Error(err))
		return err
	}

	logger.Info("order status changed successfully")
	return nil
}

// validateStatusTransition проверяет допустимость перехода между статусами
func (s *OrderService) validateStatusTransition(from, to models.OrderStatus) error {
	validTransitions := map[models.OrderStatus][]models.OrderStatus{
		models.OrderStatusNew: {
			models.OrderStatusApproved,
			models.OrderStatusRejected,
			models.OrderStatusCancelled,
		},
		models.OrderStatusApproved: {
			models.OrderStatusAssigned,
			models.OrderStatusOnHold,
			models.OrderStatusCancelled,
		},
		models.OrderStatusRejected: {
			models.OrderStatusNew, // Можно вернуть на рассмотрение
		},
		models.OrderStatusAssigned: {
			models.OrderStatusInProgress,
			models.OrderStatusOnHold,
			models.OrderStatusCancelled,
		},
		models.OrderStatusInProgress: {
			models.OrderStatusOnHold,
			models.OrderStatusCompleted,
		},
		models.OrderStatusOnHold: {
			models.OrderStatusInProgress,
			models.OrderStatusAssigned,
			models.OrderStatusCancelled,
		},
		models.OrderStatusCompleted: {},
		models.OrderStatusCancelled: {},
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return fmt.Errorf("недопустимый исходный статус: %s", from)
	}

	for _, allowedStatus := range allowed {
		if allowedStatus == to {
			return nil
		}
	}

	return fmt.Errorf("переход из статуса %s в %s недопустим", from, to)
}

// ApproveOrder одобряет заявку (менеджером)
func (s *OrderService) ApproveOrder(ctx context.Context, id, userID, userName, comment string) error {
	return s.ChangeOrderStatus(ctx, id, string(models.OrderStatusApproved), userID, userName, comment)
}

// RejectOrder отклоняет заявку
func (s *OrderService) RejectOrder(ctx context.Context, id, userID, userName, comment string) error {
	return s.ChangeOrderStatus(ctx, id, string(models.OrderStatusRejected), userID, userName, comment)
}

// AssignOrder назначает ответственного менеджера за заявку
func (s *OrderService) AssignOrder(ctx context.Context, id, assignedTo, userID, userName, comment string) error {
	logger := loggerWith(ctx, s.log,
		zap.String("order_id", id),
		zap.String("assigned_to", assignedTo),
		zap.String("operation", "AssignOrder"))
	logger.Info("assigning order to manager")

	// Проверяем, существует ли пользователь
	user, err := s.repo.GetUserByID(ctx, assignedTo)
	if err != nil {
		logger.Error("user not found", zap.Error(err))
		return ErrUserNotFound
	}

	// Проверяем роль пользователя (должен быть менеджером или админом)
	if user.Role != models.RoleManager && user.Role != models.RoleAdmin {
		logger.Error("user has invalid role for assignment")
		return fmt.Errorf("пользователь не может быть назначен ответственным")
	}

	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		logger.Error("failed to get order", zap.Error(err))
		return err
	}

	// Можно назначать только заявки в статусе approved
	if order.Status != models.OrderStatusApproved {
		logger.Error("cannot assign order with status", zap.String("status", string(order.Status)))
		return fmt.Errorf("можно назначить ответственного только для заявки в статусе approved")
	}

	if err := s.repo.AssignOrder(ctx, id, assignedTo, userID, userName, comment); err != nil {
		logger.Error("failed to assign order", zap.Error(err))
		return err
	}

	logger.Info("order assigned successfully")
	return nil
}

// AssignOrderItem назначает исполнителя для позиции заявки
func (s *OrderService) AssignOrderItem(ctx context.Context, itemID, assignedTo, userID, userName, comment string) error {
	logger := loggerWith(ctx, s.log,
		zap.String("item_id", itemID),
		zap.String("assigned_to", assignedTo),
		zap.String("operation", "AssignOrderItem"))
	logger.Info("assigning order item to executor")

	// Проверяем, существует ли пользователь
	user, err := s.repo.GetUserByID(ctx, assignedTo)
	if err != nil {
		logger.Error("user not found", zap.Error(err))
		return ErrUserNotFound
	}

	// Проверяем роль пользователя (должен быть инженером, техником или админом)
	if user.Role != models.RoleEngineer && user.Role != models.RoleTechnician && user.Role != models.RoleAdmin {
		logger.Error("user has invalid role for assignment")
		return fmt.Errorf("пользователь не может быть назначен исполнителем")
	}

	item, err := s.repo.GetItemByID(ctx, itemID)
	if err != nil {
		logger.Error("failed to get order item", zap.Error(err))
		return err
	}

	if err := s.repo.AssignOrderItem(ctx, itemID, assignedTo, userID, userName, comment); err != nil {
		logger.Error("failed to assign order item", zap.Error(err))
		return err
	}

	logger.Info("order item assigned successfully")
	return nil
}

// CompleteOrder завершает заявку
func (s *OrderService) CompleteOrder(ctx context.Context, id, userID, userName, comment string) error {
	return s.ChangeOrderStatus(ctx, id, string(models.OrderStatusCompleted), userID, userName, comment)
}

// CancelOrder отменяет заявку
func (s *OrderService) CancelOrder(ctx context.Context, id, userID, userName, comment string) error {
	return s.ChangeOrderStatus(ctx, id, string(models.OrderStatusCancelled), userID, userName, comment)
}

// PutOnHold приостанавливает заявку
func (s *OrderService) PutOnHold(ctx context.Context, id, userID, userName, comment string) error {
	return s.ChangeOrderStatus(ctx, id, string(models.OrderStatusOnHold), userID, userName, comment)
}

// UpdateOrderItem обновляет позицию заявки
func (s *OrderService) UpdateOrderItem(ctx context.Context, id string, req models.UpdateOrderItemRequest) (models.OrderItem, error) {
	logger := loggerWith(ctx, s.log, zap.String("item_id", id), zap.String("operation", "UpdateOrderItem"))
	logger.Debug("updating order item")

	item, err := s.repo.UpdateItem(ctx, id, req)
	if err != nil {
		logger.Error("failed to update item", zap.Error(err))
		return models.OrderItem{}, err
	}

	if err := s.repo.RecalculateTotal(ctx, item.OrderID); err != nil {
		logger.Warn("failed to recalculate total", zap.Error(err))
	}

	logger.Info("order item updated successfully")
	return item, nil
}

// DeleteOrderItem удаляет позицию заявки
func (s *OrderService) DeleteOrderItem(ctx context.Context, id string) error {
	logger := loggerWith(ctx, s.log, zap.String("item_id", id), zap.String("operation", "DeleteOrderItem"))
	logger.Debug("deleting order item")

	item, err := s.repo.GetItemByID(ctx, id)
	if err != nil {
		logger.Error("failed to get item", zap.Error(err))
		return err
	}

	if err := s.repo.DeleteItem(ctx, id); err != nil {
		logger.Error("failed to delete item", zap.Error(err))
		return err
	}

	if err := s.repo.RecalculateTotal(ctx, item.OrderID); err != nil {
		logger.Warn("failed to recalculate total", zap.Error(err))
	}

	logger.Info("order item deleted successfully")
	return nil
}

// DeleteOrder удаляет заявку (мягкое удаление)
func (s *OrderService) DeleteOrder(ctx context.Context, id string) error {
	logger := loggerWith(ctx, s.log, zap.String("order_id", id), zap.String("operation", "DeleteOrder"))
	logger.Info("soft deleting order")

	return s.repo.Delete(ctx, id)
}

// AddSampleToOrderItem добавляет информацию о доставке образца
func (s *OrderService) AddSampleToOrderItem(ctx context.Context, itemID string, sampleDelivered bool, sampleCount int, notes string) error {
	logger := loggerWith(ctx, s.log, zap.String("item_id", itemID), zap.String("operation", "AddSampleToOrderItem"))
	logger.Info("updating sample delivery info")

	req := models.UpdateOrderItemRequest{
		SampleDelivered: &sampleDelivered,
		SampleCount:     &sampleCount,
		SampleNotes:     &notes,
	}

	_, err := s.repo.UpdateItem(ctx, itemID, req)
	if err != nil {
		logger.Error("failed to update sample info", zap.Error(err))
		return err
	}

	logger.Info("sample delivery info updated successfully")
	return nil
}

// GetOrderItems получает все позиции заявки
func (s *OrderService) GetOrderItems(ctx context.Context, orderID string) ([]models.OrderItem, error) {
	logger := loggerWith(ctx, s.log, zap.String("order_id", orderID), zap.String("operation", "GetOrderItems"))
	logger.Debug("fetching order items")

	return s.repo.GetItemsByOrderID(ctx, orderID)
}

// CreateOrderItem создает позицию в заявке
func (s *OrderService) CreateOrderItem(ctx context.Context, orderID string, req models.CreateOrderItemRequest) (models.OrderItem, error) {
	logger := loggerWith(ctx, s.log, zap.String("order_id", orderID), zap.String("operation", "CreateOrderItem"))
	logger.Info("creating order item")

	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		logger.Error("failed to get order", zap.Error(err))
		return models.OrderItem{}, err
	}

	// Нельзя добавлять позиции в завершенные или отмененные заявки
	if order.Status == models.OrderStatusCompleted || order.Status == models.OrderStatusCancelled {
		logger.Error("cannot add items to completed/cancelled order")
		return models.OrderItem{}, fmt.Errorf("нельзя добавлять позиции в завершенную или отмененную заявку")
	}

	itemID := uuid.New().String()
	now := time.Now()

	item := models.OrderItem{
		ID:             itemID,
		OrderID:        orderID,
		TestMethodID:   req.TestMethodID,
		Quantity:       req.Quantity,
		UnitPrice:      req.UnitPrice,
		SampleRequired: req.SampleRequired,
		SampleNotes:    req.SampleNotes,
		SampleCount:    req.SampleCount,
		Status:         models.OrderItemStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	item.Subtotal = float64(item.Quantity) * item.UnitPrice

	if err := s.repo.CreateItem(ctx, item); err != nil {
		logger.Error("failed to create order item", zap.Error(err))
		return models.OrderItem{}, err
	}

	if err := s.repo.RecalculateTotal(ctx, orderID); err != nil {
		logger.Warn("failed to recalculate total", zap.Error(err))
	}

	logger.Info("order item created successfully", zap.String("item_id", itemID))
	return item, nil
}

// GetOrderItem получает позицию заявки по ID
func (s *OrderService) GetOrderItem(ctx context.Context, itemID string) (models.OrderItem, error) {
	logger := loggerWith(ctx, s.log, zap.String("item_id", itemID), zap.String("operation", "GetOrderItem"))
	logger.Debug("fetching order item")

	return s.repo.GetItemByID(ctx, itemID)
}

// GetOrderWorkflow получает историю изменения статусов заявки
func (s *OrderService) GetOrderWorkflow(ctx context.Context, orderID string) ([]models.OrderWorkflowEntry, error) {
	logger := loggerWith(ctx, s.log, zap.String("order_id", orderID), zap.String("operation", "GetOrderWorkflow"))
	logger.Debug("fetching order workflow")

	return s.repo.GetWorkflowByOrderID(ctx, orderID)
}
