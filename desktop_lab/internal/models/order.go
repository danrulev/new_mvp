package models

import "time"

// OrderStatus представляет статус заявки на исследования
type OrderStatus string

const (
	OrderStatusNew       OrderStatus = "new"        // Новая заявка, ожидает рассмотрения
	OrderStatusApproved  OrderStatus = "approved"   // Заявка одобрена, готова к назначению исполнителей
	OrderStatusRejected  OrderStatus = "rejected"   // Заявка отклонена
	OrderStatusAssigned  OrderStatus = "assigned"   // Назначен ответственный исполнитель
	OrderStatusInProgress OrderStatus = "in_progress" // Исследования в процессе
	OrderStatusOnHold    OrderStatus = "on_hold"    // Приостановлена (ожидание образцов, уточнений)
	OrderStatusCompleted OrderStatus = "completed"  // Исследования завершены
	OrderStatusCancelled OrderStatus = "cancelled"  // Заявка отменена
)

// IsValid проверяет, является ли статус допустимым
func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusNew, OrderStatusApproved, OrderStatusRejected,
		OrderStatusAssigned, OrderStatusInProgress, OrderStatusOnHold,
		OrderStatusCompleted, OrderStatusCancelled:
		return true
	default:
		return false
	}
}

// OrderItemStatus представляет статус позиции заявки
type OrderItemStatus string

const (
	OrderItemStatusPending   OrderItemStatus = "pending"    // Ожидает обработки
	OrderItemStatusAssigned  OrderItemStatus = "assigned"   // Назначен исполнитель
	OrderItemStatusInWork    OrderItemStatus = "in_work"    // В работе у исполнителя
	OrderItemStatusCompleted OrderItemStatus = "completed"  // Выполнено
	OrderItemStatusCancelled OrderItemStatus = "cancelled"  // Отменено
)

// IsValid проверяет, является ли статус позиции допустимым
func (s OrderItemStatus) IsValid() bool {
	switch s {
	case OrderItemStatusPending, OrderItemStatusAssigned, OrderItemStatusInWork,
		OrderItemStatusCompleted, OrderItemStatusCancelled:
		return true
	default:
		return false
	}
}

// Order представляет заявку на лабораторные исследования
type Order struct {
	ID              string      `json:"id" db:"id"`
	CreatedBy       string      `json:"created_by" db:"created_by"` // Кто создал заявку
	AssignedTo      *string     `json:"assigned_to,omitempty" db:"assigned_to"` // Ответственный менеджер
	Status          OrderStatus `json:"status" db:"status"`
	Priority        string      `json:"priority" db:"priority"` // normal, high, urgent
	Title           string      `json:"title" db:"title"`
	Description     string      `json:"description" db:"description"`
	InternalComment string      `json:"internal_comment,omitempty" db:"internal_comment"` // Внутренний комментарий для сотрудников
	ExternalComment string      `json:"external_comment,omitempty" db:"external_comment"` // Комментарий для клиента (если есть)
	TotalAmount     float64     `json:"total_amount" db:"total_amount"`
	Currency        string      `json:"currency" db:"currency"`
	ClientName      string      `json:"client_name" db:"client_name"`
	ClientEmail     string      `json:"client_email" db:"client_email"`
	ClientPhone     string      `json:"client_phone" db:"client_phone"`
	SampleLocation  string      `json:"sample_location,omitempty" db:"sample_location"` // Где находятся образцы
	DueDate         *time.Time  `json:"due_date,omitempty" db:"due_date"` // Плановая дата завершения
	CreatedAt       time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at" db:"updated_at"`
	CompletedAt     *time.Time  `json:"completed_at,omitempty" db:"completed_at"`
}

// OrderItem представляет позицию в заявке (конкретное исследование)
type OrderItem struct {
	ID               string          `json:"id" db:"id"`
	OrderID          string          `json:"order_id" db:"order_id"`
	TestMethodID     string          `json:"test_method_id" db:"test_method_id"`
	TestMethodName   string          `json:"test_method_name" db:"test_method_name"`
	AssignedTo       *string         `json:"assigned_to,omitempty" db:"assigned_to"` // Исполнитель исследования
	Quantity         int             `json:"quantity" db:"quantity"`
	UnitPrice        float64         `json:"unit_price" db:"unit_price"`
	Subtotal         float64         `json:"subtotal" db:"subtotal"`
	Status           OrderItemStatus `json:"status" db:"status"`
	SampleRequired   bool            `json:"sample_required" db:"sample_required"`
	SampleNotes      string          `json:"sample_notes" db:"sample_notes"`
	SampleCount      int             `json:"sample_count" db:"sample_count"`
	SampleDelivered  bool            `json:"sample_delivered" db:"sample_delivered"`
	SampleReceivedAt *time.Time      `json:"sample_received_at,omitempty" db:"sample_received_at"`
	InternalNotes    string          `json:"internal_notes,omitempty" db:"internal_notes"`
	ResultNotes      string          `json:"result_notes,omitempty" db:"result_notes"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
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
	Title           string                   `json:"title" validate:"required"`
	Description     string                   `json:"description" validate:"required"`
	Priority        string                   `json:"priority"` // normal, high, urgent
	ClientName      string                   `json:"client_name" validate:"required"`
	ClientEmail     string                   `json:"client_email" validate:"required,email"`
	ClientPhone     string                   `json:"client_phone"`
	ExternalComment string                   `json:"external_comment"`
	InternalComment string                   `json:"internal_comment"`
	SampleLocation  string                   `json:"sample_location"`
	DueDate         *time.Time               `json:"due_date,omitempty"`
	Items           []CreateOrderItemRequest `json:"items" validate:"required,min=1,dive"`
}

// CreateOrderItemRequest представляет позицию для создания в заявке
type CreateOrderItemRequest struct {
	TestMethodID   string  `json:"test_method_id" validate:"required"`
	Quantity       int     `json:"quantity" validate:"required,gte=1"`
	UnitPrice      float64 `json:"unit_price"`
	SampleRequired bool    `json:"sample_required"`
	SampleNotes    string  `json:"sample_notes"`
	SampleCount    int     `json:"sample_count"`
}

// UpdateOrderRequest представляет запрос на обновление заявки
type UpdateOrderRequest struct {
	Title           *string    `json:"title,omitempty"`
	Description     *string    `json:"description,omitempty"`
	Priority        *string    `json:"priority,omitempty"`
	ClientName      *string    `json:"client_name,omitempty"`
	ClientEmail     *string    `json:"client_email,omitempty"`
	ClientPhone     *string    `json:"client_phone,omitempty"`
	ExternalComment *string    `json:"external_comment,omitempty"`
	InternalComment *string    `json:"internal_comment,omitempty"`
	SampleLocation  *string    `json:"sample_location,omitempty"`
	DueDate         *time.Time `json:"due_date,omitempty"`
}

// AssignOrderRequest представляет запрос на назначение ответственного за заявку
type AssignOrderRequest struct {
	AssignedTo string `json:"assigned_to" validate:"required"`
	Comment    string `json:"comment"`
}

// AssignOrderItemRequest представляет запрос на назначение исполнителя позиции
type AssignOrderItemRequest struct {
	AssignedTo string `json:"assigned_to" validate:"required"`
	Comment    string `json:"comment"`
}

// UpdateOrderItemRequest представляет запрос на обновление позиции заявки
type UpdateOrderItemRequest struct {
	Quantity         *int       `json:"quantity,omitempty"`
	UnitPrice        *float64   `json:"unit_price,omitempty"`
	SampleNotes      *string    `json:"sample_notes,omitempty"`
	SampleCount      *int       `json:"sample_count,omitempty"`
	SampleDelivered  *bool      `json:"sample_delivered,omitempty"`
	InternalNotes    *string    `json:"internal_notes,omitempty"`
	ResultNotes      *string    `json:"result_notes,omitempty"`
	Status           *OrderItemStatus `json:"status,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

// OrderStatusChangeRequest представляет запрос на изменение статуса заявки
type OrderStatusChangeRequest struct {
	Status  OrderStatus `json:"status" validate:"required"`
	Comment string      `json:"comment"`
}

// OrderListFilter представляет фильтры для списка заявок
type OrderListFilter struct {
	Paginated
	CreatedBy      string      `form:"created_by"`
	AssignedTo     string      `form:"assigned_to"`
	Status         OrderStatus `form:"status"`
	Priority       string      `form:"priority"`
	DateFrom       string      `form:"date_from"`
	DateTo         string      `form:"date_to"`
	SearchQuery    string      `form:"search"` // Поиск по title, description, client_name
}

// OrderResponse представляет полный ответ по заявке
type OrderResponse struct {
	Order    Order                `json:"order"`
	Items    []OrderItem          `json:"items"`
	Workflow []OrderWorkflowEntry `json:"workflow"`
	Meta     PaginatedMetadata    `json:"meta,omitempty"`
}

// OrderListResponse представляет ответ со списком заявок
type OrderListResponse struct {
	Orders []Order           `json:"orders"`
	Meta   PaginatedMetadata `json:"meta"`
}

// OrderItemResponse представляет ответ по позиции заявки
type OrderItemResponse struct {
	Item OrderItem `json:"item"`
}

// OrderItemListResponse представляет ответ со списком позиций заявки
type OrderItemListResponse struct {
	Items []OrderItem       `json:"items"`
	Meta  PaginatedMetadata `json:"meta"`
}
