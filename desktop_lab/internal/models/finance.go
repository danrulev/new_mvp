package models

import "time"

// InvoiceStatus представляет статус счета
type InvoiceStatus string

const (
	InvoiceStatusDraft    InvoiceStatus = "draft"
	InvoiceStatusSent     InvoiceStatus = "sent"
	InvoiceStatusPaid     InvoiceStatus = "paid"
	InvoiceStatusOverdue  InvoiceStatus = "overdue"
	InvoiceStatusCancelled InvoiceStatus = "cancelled"
)

// IsValid проверяет, является ли статус допустимым
func (s InvoiceStatus) IsValid() bool {
	switch s {
	case InvoiceStatusDraft, InvoiceStatusSent, InvoiceStatusPaid,
		InvoiceStatusOverdue, InvoiceStatusCancelled:
		return true
	default:
		return false
	}
}

// PaymentMethod представляет способ оплаты
type PaymentMethod string

const (
	PaymentMethodCard        PaymentMethod = "card"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodCash        PaymentMethod = "cash"
	PaymentMethodOnline      PaymentMethod = "online"
)

// PriceList представляет прайс-лист организации
type PriceList struct {
	ID             string     `json:"id" db:"id"`
	OrganizationID string     `json:"organization_id" db:"organization_id"`
	Name           string     `json:"name" db:"name"`
	IsDefault      bool       `json:"is_default" db:"is_default"`
	ValidFrom      *time.Time `json:"valid_from,omitempty" db:"valid_from"`
	ValidTo        *time.Time `json:"valid_to,omitempty" db:"valid_to"`
	Currency       string     `json:"currency" db:"currency"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// PriceListItem представляет позицию прайс-листа
type PriceListItem struct {
	ID              string  `json:"id" db:"id"`
	PriceListID     string  `json:"price_list_id" db:"price_list_id"`
	TestMethodID    string  `json:"test_method_id" db:"test_method_id"`
	BasePrice       float64 `json:"base_price" db:"base_price"`
	DiscountPercent float64 `json:"discount_percent" db:"discount_percent"`
	MinQuantity     int     `json:"min_quantity" db:"min_quantity"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// Invoice представляет счет на оплату
type Invoice struct {
	ID            string        `json:"id" db:"id"`
	OrderID       string        `json:"order_id" db:"order_id"`
	InvoiceNumber string        `json:"invoice_number" db:"invoice_number"`
	CustomerID    string        `json:"customer_id" db:"customer_id"`
	BillingAddress JSONStringMap `json:"billing_address" db:"billing_address"`
	Subtotal      float64       `json:"subtotal" db:"subtotal"`
	Tax           float64       `json:"tax" db:"tax"`
	Discount      float64       `json:"discount" db:"discount"`
	Total         float64       `json:"total" db:"total"`
	Status        InvoiceStatus `json:"status" db:"status"`
	DueDate       *time.Time    `json:"due_date,omitempty" db:"due_date"`
	PaidAt        *time.Time    `json:"paid_at,omitempty" db:"paid_at"`
	CreatedAt     time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at" db:"updated_at"`
}

// InvoiceItem представляет позицию счета
type InvoiceItem struct {
	ID            string  `json:"id" db:"id"`
	InvoiceID     string  `json:"invoice_id" db:"invoice_id"`
	OrderItemID   string  `json:"order_item_id" db:"order_item_id"`
	Description   string  `json:"description" db:"description"`
	Quantity      int     `json:"quantity" db:"quantity"`
	UnitPrice     float64 `json:"unit_price" db:"unit_price"`
	Subtotal      float64 `json:"subtotal" db:"subtotal"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// Payment представляет платеж по счету
type Payment struct {
	ID            string          `json:"id" db:"id"`
	InvoiceID     string          `json:"invoice_id" db:"invoice_id"`
	Amount        float64         `json:"amount" db:"amount"`
	PaymentMethod PaymentMethod   `json:"payment_method" db:"payment_method"`
	TransactionID string          `json:"transaction_id" db:"transaction_id"`
	PaymentDate   time.Time       `json:"payment_date" db:"payment_date"`
	Metadata      JSONStringMap   `json:"metadata" db:"metadata"`
	Status        string          `json:"status" db:"status"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
}

// PaymentGateway представляет платежный шлюз
type PaymentGateway struct {
	ID         string        `json:"id" db:"id"`
	Provider   string        `json:"provider" db:"provider"` // 'stripe', 'cloudpayments', 'tinkoff'
	APIKeys    string        `json:"-" db:"api_keys"` // Encrypted
	WebhookURL string        `json:"webhook_url" db:"webhook_url"`
	IsActive   bool          `json:"is_active" db:"is_active"`
	Config     JSONStringMap `json:"config,omitempty" db:"config"`
	CreatedAt  time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at" db:"updated_at"`
}

// CreatePriceListRequest представляет запрос на создание прайс-листа
type CreatePriceListRequest struct {
	OrganizationID string     `json:"organization_id" validate:"required"`
	Name           string     `json:"name" validate:"required"`
	IsDefault      bool       `json:"is_default"`
	ValidFrom      *string    `json:"valid_from,omitempty"`
	ValidTo        *string    `json:"valid_to,omitempty"`
	Currency       string     `json:"currency" validate:"required,len=3"`
}

// CreatePriceListItemRequest представляет запрос на создание позиции прайс-листа
type CreatePriceListItemRequest struct {
	PriceListID     string  `json:"price_list_id" validate:"required"`
	TestMethodID    string  `json:"test_method_id" validate:"required"`
	BasePrice       float64 `json:"base_price" validate:"required,gt=0"`
	DiscountPercent float64 `json:"discount_percent" validate:"gte=0,lte=100"`
	MinQuantity     int     `json:"min_quantity" validate:"gte=1"`
}

// CreateInvoiceRequest представляет запрос на создание счета
type CreateInvoiceRequest struct {
	OrderID        string            `json:"order_id" validate:"required"`
	CustomerID     string            `json:"customer_id" validate:"required"`
	BillingAddress JSONStringMap     `json:"billing_address"`
	DueDate        *string           `json:"due_date,omitempty"`
	Discount       float64           `json:"discount" validate:"gte=0"`
	TaxRate        float64           `json:"tax_rate" validate:"gte=0"`
}

// CreatePaymentRequest представляет запрос на создание платежа
type CreatePaymentRequest struct {
	InvoiceID     string          `json:"invoice_id" validate:"required"`
	Amount        float64         `json:"amount" validate:"required,gt=0"`
	PaymentMethod PaymentMethod   `json:"payment_method" validate:"required"`
	TransactionID string          `json:"transaction_id"`
	Metadata      JSONStringMap   `json:"metadata,omitempty"`
}

// InvoiceResponse представляет полный ответ по счету
type InvoiceResponse struct {
	Invoice Invoice       `json:"invoice"`
	Items   []InvoiceItem `json:"items"`
	Payments []Payment    `json:"payments"`
}

// InvoiceListFilter представляет фильтры для списка счетов
type InvoiceListFilter struct {
	Paginated
	OrderID      string        `form:"order_id"`
	CustomerID   string        `form:"customer_id"`
	Status       InvoiceStatus `form:"status"`
	DateFrom     string        `form:"date_from"`
	DateTo       string        `form:"date_to"`
}

// InvoiceListResponse представляет ответ со списком счетов
type InvoiceListResponse struct {
	Invoices []Invoice         `json:"invoices"`
	Meta     PaginatedMetadata `json:"meta"`
}

// PaymentListResponse представляет ответ со списком платежей
type PaymentListResponse struct {
	Payments []Payment         `json:"payments"`
	Meta     PaginatedMetadata `json:"meta"`
}

// PaymentGatewayResponse представляет ответ по платежному шлюзу
type PaymentGatewayResponse struct {
	Gateway PaymentGateway `json:"gateway"`
}

// PaymentGatewayListResponse представляет ответ со списком платежных шлюзов
type PaymentGatewayListResponse struct {
	Gateways []PaymentGateway `json:"gateways"`
}
