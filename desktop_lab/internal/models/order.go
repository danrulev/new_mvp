package models

import "time"

// OrderStatus представляет статус заявки
type OrderStatus string

const (
	OrderStatusDraft      OrderStatus = "draft"
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusAccepted   OrderStatus = "accepted"
	OrderStatusRejected   OrderStatus = "rejected"
	OrderStatusInProgress OrderStatus = "in_progress"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

// IsValid проверяет, является ли статус допустимым
func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusDraft, OrderStatusPending, OrderStatusAccepted,
		OrderStatusRejected, OrderStatusInProgress,
		OrderStatusCompleted, OrderStatusCancelled:
		return true
	default:
		return false
	}
}

// Order представляет заявку клиента на лабораторные испытания
type Order struct {
	ID             string      `json:"id" db:"id"`
	CustomerID     string      `json:"customer_id" db:"customer_id"`
	OrganizationID string      `json:"organization_id" db:"organization_id"`
	Status         OrderStatus `json:"status" db:"status"`
	TotalAmount    float64     `json:"total_amount" db:"total_amount"`
	Currency       string      `json:"currency" db:"currency"`
	CustomerName   string      `json:"customer_name" db:"customer_name"`
	CustomerEmail  string      `json:"customer_email" db:"customer_email"`
	CustomerPhone  string      `json:"customer_phone" db:"customer_phone"`
	Comment        string      `json:"comment" db:"comment"`
	CreatedAt      time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at" db:"updated_at"`
	CompletedAt    *time.Time  `json:"completed_at,omitempty" db:"completed_at"`
}

// OrderItem представляет позицию в заявке (конкретное исследование)
type OrderItem struct {
	ID              string  `json:"id" db:"id"`
	OrderID         string  `json:"order_id" db:"order_id"`
	TestMethodID    string  `json:"test_method_id" db:"test_method_id"`
	TestMethodName  string  `json:"test_method_name" db:"test_method_name"`
	Quantity        int     `json:"quantity" db:"quantity"`
	UnitPrice       float64 `json:"unit_price" db:"unit_price"`
	Subtotal        float64 `json:"subtotal" db:"subtotal"`
	Status          string  `json:"status" db:"status"`
	SampleRequired  bool    `json:"sample_required" db:"sample_required"`
	SampleNotes     string  `json:"sample_notes" db:"sample_notes"`
	SampleCount     int     `json:"sample_count" db:"sample_count"`
	SampleDelivered bool    `json:"sample_delivered" db:"sample_delivered"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// OrderWorkflowEntry представляет запись в истории изменения статусов заявки
type OrderWorkflowEntry struct {
	ID         string    `json:"id" db:"id"`
	OrderID    string    `json:"order_id" db:"order_id"`
	FromStatus string    `json:"from_status" db:"from_status"`
	ToStatus   string    `json:"to_status" db:"to_status"`
	UserID     string    `json:"user_id" db:"user_id"`
	UserName   string    `json:"user_name" db:"user_name"`
	Comment    string    `json:"comment" db:"comment"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// CreateOrderRequest представляет запрос на создание заявки
type CreateOrderRequest struct {
	OrganizationID string              `json:"organization_id" validate:"required"`
	CustomerName   string              `json:"customer_name" validate:"required"`
	CustomerEmail  string              `json:"customer_email" validate:"required,email"`
	CustomerPhone  string              `json:"customer_phone"`
	Comment        string              `json:"comment"`
	Items          []CreateOrderItemRequest `json:"items" validate:"required,min=1,dive"`
}

// CreateOrderItemRequest представляет позицию для создания в заявке
type CreateOrderItemRequest struct {
	TestMethodID   string  `json:"test_method_id" validate:"required"`
	Quantity       int     `json:"quantity" validate:"required,gte=1"`
	SampleRequired bool    `json:"sample_required"`
	SampleNotes    string  `json:"sample_notes"`
	SampleCount    int     `json:"sample_count"`
}

// UpdateOrderRequest представляет запрос на обновление заявки
type UpdateOrderRequest struct {
	CustomerName  *string `json:"customer_name,omitempty"`
	CustomerEmail *string `json:"customer_email,omitempty"`
	CustomerPhone *string `json:"customer_phone,omitempty"`
	Comment       *string `json:"comment,omitempty"`
}

// UpdateOrderItemRequest представляет запрос на обновление позиции заявки
type UpdateOrderItemRequest struct {
	Quantity       *int    `json:"quantity,omitempty"`
	SampleNotes    *string `json:"sample_notes,omitempty"`
	SampleCount    *int    `json:"sample_count,omitempty"`
	SampleDelivered *bool  `json:"sample_delivered,omitempty"`
}

// OrderStatusChangeRequest представляет запрос на изменение статуса заявки
type OrderStatusChangeRequest struct {
	Status  OrderStatus `json:"status" validate:"required"`
	Comment string      `json:"comment"`
}

// OrderListFilter представляет фильтры для списка заявок
type OrderListFilter struct {
	Paginated
	OrganizationID string      `form:"organization_id"`
	CustomerID     string      `form:"customer_id"`
	Status         OrderStatus `form:"status"`
	DateFrom       string      `form:"date_from"`
	DateTo         string      `form:"date_to"`
}

// OrderResponse представляет полный ответ по заявке
type OrderResponse struct {
	Order       Order              `json:"order"`
	Items       []OrderItem        `json:"items"`
	Workflow    []OrderWorkflowEntry `json:"workflow"`
	Meta        PaginatedMetadata  `json:"meta,omitempty"`
}

// OrderListResponse представляет ответ со списком заявок
type OrderListResponse struct {
	Orders []Order             `json:"orders"`
	Meta   PaginatedMetadata   `json:"meta"`
}

// OrderItemResponse представляет ответ по позиции заявки
type OrderItemResponse struct {
	Item OrderItem `json:"item"`
}

// OrderItemListResponse представляет ответ со списком позиций заявки
type OrderItemListResponse struct {
	Items []OrderItem         `json:"items"`
	Meta  PaginatedMetadata   `json:"meta"`
}
