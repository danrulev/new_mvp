package service

import (
	"context"
	"desktop_lab/internal/models"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

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
}

type OrderService struct {
	repo         OrderRepo
	orgTestsRepo OrganizationTestsRepo
	log          *zap.Logger
}

func NewOrderService(repo OrderRepo, orgTestsRepo OrganizationTestsRepo, log *zap.Logger) *OrderService {
	return &OrderService{
		repo:         repo,
		orgTestsRepo: orgTestsRepo,
		log:          log,
	}
}

// CreateOrder создает новую заявку
func (s *OrderService) CreateOrder(ctx context.Context, req models.CreateOrderRequest, userID string) (models.Order, error) {
	logger := loggerWith(ctx, s.log, zap.String("operation", "CreateOrder"))
	logger.Info("creating new order")

	orderID := uuid.New().String()

	order := models.Order{
		ID:             orderID,
		CustomerID:     userID,
		OrganizationID: req.OrganizationID,
		Status:         models.OrderStatusDraft,
		Currency:       "RUB",
		CustomerName:   req.CustomerName,
		CustomerEmail:  req.CustomerEmail,
		CustomerPhone:  req.CustomerPhone,
		Comment:        req.Comment,
	}

	if err := s.repo.Create(ctx, order); err != nil {
		logger.Error("failed to create order", zap.Error(err))
		return models.Order{}, err
	}

	// Создаем позиции заявки
	for _, itemReq := range req.Items {
		orgTest, err := s.orgTestsRepo.GetOrganizationTest(ctx, itemReq.TestMethodID)
		if err != nil {
			logger.Error("failed to get test method", zap.Error(err))
			return models.Order{}, fmt.Errorf("не удалось получить информацию о тесте: %w", err)
		}

		item := models.OrderItem{
			ID:             uuid.New().String(),
			OrderID:        orderID,
			TestMethodID:   itemReq.TestMethodID,
			TestMethodName: orgTest.Description,
			Quantity:       itemReq.Quantity,
			UnitPrice:      orgTest.Price,
			SampleRequired: itemReq.SampleRequired,
			SampleNotes:    itemReq.SampleNotes,
			SampleCount:    itemReq.SampleCount,
			Status:         "pending",
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

	logger.Info("order created successfully", zap.String("order_id", orderID))
	return s.repo.GetByID(ctx, orderID)
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
		models.OrderStatusDraft: {
			models.OrderStatusPending,
			models.OrderStatusCancelled,
		},
		models.OrderStatusPending: {
			models.OrderStatusAccepted,
			models.OrderStatusRejected,
			models.OrderStatusCancelled,
		},
		models.OrderStatusAccepted: {
			models.OrderStatusInProgress,
			models.OrderStatusCancelled,
		},
		models.OrderStatusRejected: {
			models.OrderStatusPending,
		},
		models.OrderStatusInProgress: {
			models.OrderStatusCompleted,
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

// AcceptOrder принимает заявку
func (s *OrderService) AcceptOrder(ctx context.Context, id, userID, userName, comment string) error {
	return s.ChangeOrderStatus(ctx, id, string(models.OrderStatusAccepted), userID, userName, comment)
}

// RejectOrder отклоняет заявку
func (s *OrderService) RejectOrder(ctx context.Context, id, userID, userName, comment string) error {
	return s.ChangeOrderStatus(ctx, id, string(models.OrderStatusRejected), userID, userName, comment)
}

// CompleteOrder завершает заявку
func (s *OrderService) CompleteOrder(ctx context.Context, id, userID, userName, comment string) error {
	return s.ChangeOrderStatus(ctx, id, string(models.OrderStatusCompleted), userID, userName, comment)
}

// CancelOrder отменяет заявку
func (s *OrderService) CancelOrder(ctx context.Context, id, userID, userName, comment string) error {
	return s.ChangeOrderStatus(ctx, id, string(models.OrderStatusCancelled), userID, userName, comment)
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

	orgTest, err := s.orgTestsRepo.GetOrganizationTest(ctx, req.TestMethodID)
	if err != nil {
		logger.Error("failed to get test method", zap.Error(err))
		return models.OrderItem{}, fmt.Errorf("не удалось получить информацию о тесте: %w", err)
	}

	itemID := uuid.New().String()
	item := models.OrderItem{
		ID:             itemID,
		OrderID:        orderID,
		TestMethodID:   req.TestMethodID,
		TestMethodName: orgTest.Description,
		Quantity:       req.Quantity,
		UnitPrice:      orgTest.Price,
		SampleRequired: req.SampleRequired,
		SampleNotes:    req.SampleNotes,
		SampleCount:    req.SampleCount,
		Status:         "pending",
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
