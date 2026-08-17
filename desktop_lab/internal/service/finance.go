package service

import (
"context"
"desktop_lab/internal/models"
"fmt"
"time"

"github.com/google/uuid"
"go.uber.org/zap"
)

// PriceListService сервис для работы с прайс-листами
type PriceListService struct {
repo         PriceListRepoInterface
orgTestsRepo OrganizationTestsRepo
log          *zap.Logger
}

type PriceListRepoInterface interface {
Create(ctx context.Context, pl models.PriceList) error
GetByID(ctx context.Context, id string) (models.PriceList, error)
GetByOrganization(ctx context.Context, orgID string) ([]models.PriceList, error)
CreateItem(ctx context.Context, item models.PriceListItem) error
GetItemsByPriceList(ctx context.Context, priceListID string) ([]models.PriceListItem, error)
}

func NewPriceListService(repo PriceListRepoInterface, orgTestsRepo OrganizationTestsRepo, log *zap.Logger) *PriceListService {
return &PriceListService{
repo:         repo,
orgTestsRepo: orgTestsRepo,
log:          log,
}
}

// CreatePriceList создает новый прайс-лист
func (s *PriceListService) CreatePriceList(ctx context.Context, req models.CreatePriceListRequest) (models.PriceList, error) {
logger := loggerWith(ctx, s.log, zap.String("operation", "CreatePriceList"))
logger.Info("creating price list")

priceListID := uuid.New().String()

var validFrom, validTo *time.Time
if req.ValidFrom != nil {
t, err := time.Parse(time.RFC3339, *req.ValidFrom)
if err == nil {
validFrom = &t
}
}
if req.ValidTo != nil {
t, err := time.Parse(time.RFC3339, *req.ValidTo)
if err == nil {
validTo = &t
}
}

priceList := models.PriceList{
ID:             priceListID,
OrganizationID: req.OrganizationID,
Name:           req.Name,
IsDefault:      req.IsDefault,
ValidFrom:      validFrom,
ValidTo:        validTo,
Currency:       req.Currency,
CreatedAt:      time.Now(),
UpdatedAt:      time.Now(),
}

if err := s.repo.Create(ctx, priceList); err != nil {
logger.Error("failed to create price list", zap.Error(err))
return models.PriceList{}, err
}

logger.Info("price list created successfully", zap.String("price_list_id", priceListID))
return priceList, nil
}

// GetPriceList получает прайс-лист по ID
func (s *PriceListService) GetPriceList(ctx context.Context, id string) (models.PriceList, error) {
logger := loggerWith(ctx, s.log, zap.String("price_list_id", id), zap.String("operation", "GetPriceList"))
logger.Debug("fetching price list")

return s.repo.GetByID(ctx, id)
}

// GetPriceListsByOrganization получает прайс-листы организации
func (s *PriceListService) GetPriceListsByOrganization(ctx context.Context, orgID string) ([]models.PriceList, error) {
logger := loggerWith(ctx, s.log, zap.String("organization_id", orgID), zap.String("operation", "GetPriceListsByOrganization"))
logger.Debug("fetching price lists")

return s.repo.GetByOrganization(ctx, orgID)
}

// AddPriceListItem добавляет позицию в прайс-лист
func (s *PriceListService) AddPriceListItem(ctx context.Context, req models.CreatePriceListItemRequest) (models.PriceListItem, error) {
logger := loggerWith(ctx, s.log, zap.String("operation", "AddPriceListItem"))
logger.Info("adding price list item")

itemID := uuid.New().String()

item := models.PriceListItem{
ID:              itemID,
PriceListID:     req.PriceListID,
TestMethodID:    req.TestMethodID,
BasePrice:       req.BasePrice,
DiscountPercent: req.DiscountPercent,
MinQuantity:     req.MinQuantity,
CreatedAt:       time.Now(),
UpdatedAt:       time.Now(),
}

if err := s.repo.CreateItem(ctx, item); err != nil {
logger.Error("failed to create price list item", zap.Error(err))
return models.PriceListItem{}, err
}

logger.Info("price list item added successfully", zap.String("item_id", itemID))
return item, nil
}

// GetPriceListItems получает позиции прайс-листа
func (s *PriceListService) GetPriceListItems(ctx context.Context, priceListID string) ([]models.PriceListItem, error) {
logger := loggerWith(ctx, s.log, zap.String("price_list_id", priceListID), zap.String("operation", "GetPriceListItems"))
logger.Debug("fetching price list items")

return s.repo.GetItemsByPriceList(ctx, priceListID)
}

// InvoiceService сервис для работы со счетами
type InvoiceService struct {
repo      InvoiceRepoInterface
orderRepo OrderRepo
log       *zap.Logger
}

type InvoiceRepoInterface interface {
Create(ctx context.Context, invoice models.Invoice) error
CreateItem(ctx context.Context, item models.InvoiceItem) error
GetByID(ctx context.Context, id string) (models.Invoice, error)
GetWithItemsAndPayments(ctx context.Context, id string) (models.InvoiceResponse, error)
GetItemsByInvoiceID(ctx context.Context, invoiceID string) ([]models.InvoiceItem, error)
List(ctx context.Context, filter models.InvoiceListFilter) ([]models.Invoice, int64, error)
UpdateStatus(ctx context.Context, id string, status models.InvoiceStatus) error
}

func NewInvoiceService(repo InvoiceRepoInterface, orderRepo OrderRepo, log *zap.Logger) *InvoiceService {
return &InvoiceService{
repo:      repo,
orderRepo: orderRepo,
log:       log,
}
}

// generateInvoiceNumber генерирует номер счета
func (s *InvoiceService) generateInvoiceNumber(ctx context.Context) (string, error) {
// Простая реализация - можно усложнить с префиксом организации и датой
return fmt.Sprintf("INV-%d", time.Now().UnixNano()), nil
}

// CreateInvoice создает новый счет
func (s *InvoiceService) CreateInvoice(ctx context.Context, req models.CreateInvoiceRequest, userID string) (models.Invoice, error) {
logger := loggerWith(ctx, s.log, zap.String("operation", "CreateInvoice"))
logger.Info("creating invoice")

// Получаем заявку
order, err := s.orderRepo.GetByID(ctx, req.OrderID)
if err != nil {
logger.Error("failed to get order", zap.Error(err))
return models.Invoice{}, fmt.Errorf("не удалось получить заявку: %w", err)
}
_ = order // Используем order для избежания ошибки компиляции

// Получаем позиции заявки
orderItems, err := s.orderRepo.GetItemsByOrderID(ctx, req.OrderID)
if err != nil {
logger.Error("failed to get order items", zap.Error(err))
return models.Invoice{}, fmt.Errorf("не удалось получить позиции заявки: %w", err)
}

invoiceID := uuid.New().String()
invoiceNumber, err := s.generateInvoiceNumber(ctx)
if err != nil {
logger.Error("failed to generate invoice number", zap.Error(err))
return models.Invoice{}, err
}

// Рассчитываем суммы
var subtotal float64
for _, item := range orderItems {
subtotal += item.Subtotal
}

tax := subtotal * req.TaxRate / 100
total := subtotal + tax - req.Discount

var dueDate *time.Time
if req.DueDate != nil {
t, err := time.Parse(time.RFC3339, *req.DueDate)
if err == nil {
dueDate = &t
}
}

invoice := models.Invoice{
ID:             invoiceID,
OrderID:        req.OrderID,
InvoiceNumber:  invoiceNumber,
CustomerID:     req.CustomerID,
BillingAddress: req.BillingAddress,
Subtotal:       subtotal,
Tax:            tax,
Discount:       req.Discount,
Total:          total,
Status:         models.InvoiceStatusDraft,
DueDate:        dueDate,
CreatedAt:      time.Now(),
UpdatedAt:      time.Now(),
}

if err := s.repo.Create(ctx, invoice); err != nil {
logger.Error("failed to create invoice", zap.Error(err))
return models.Invoice{}, err
}

// Создаем позиции счета
for _, oi := range orderItems {
invoiceItem := models.InvoiceItem{
ID:          uuid.New().String(),
InvoiceID:   invoiceID,
OrderItemID: oi.ID,
Description: oi.TestMethodName,
Quantity:    oi.Quantity,
UnitPrice:   oi.UnitPrice,
Subtotal:    oi.Subtotal,
CreatedAt:   time.Now(),
}

if err := s.repo.CreateItem(ctx, invoiceItem); err != nil {
logger.Error("failed to create invoice item", zap.Error(err))
return models.Invoice{}, err
}
}

logger.Info("invoice created successfully", zap.String("invoice_id", invoiceID), zap.String("invoice_number", invoiceNumber))
return invoice, nil
}

// GetInvoice получает счет по ID
func (s *InvoiceService) GetInvoice(ctx context.Context, id string) (models.InvoiceResponse, error) {
logger := loggerWith(ctx, s.log, zap.String("invoice_id", id), zap.String("operation", "GetInvoice"))
logger.Debug("fetching invoice")

return s.repo.GetWithItemsAndPayments(ctx, id)
}

// ListInvoices получает список счетов с фильтрацией
func (s *InvoiceService) ListInvoices(ctx context.Context, filter models.InvoiceListFilter) (models.InvoiceListResponse, error) {
logger := loggerWith(ctx, s.log, zap.Any("filter", filter), zap.String("operation", "ListInvoices"))
logger.Debug("fetching invoices list")

invoices, total, err := s.repo.List(ctx, filter)
if err != nil {
logger.Error("failed to get invoices list", zap.Error(err))
return models.InvoiceListResponse{}, err
}

return models.InvoiceListResponse{
Invoices: invoices,
Meta:     models.MakePaginatedMetadata(filter.Limit, filter.Offset, total),
}, nil
}

// SendInvoice отправляет счет (меняет статус на sent)
func (s *InvoiceService) SendInvoice(ctx context.Context, id string) error {
logger := loggerWith(ctx, s.log, zap.String("invoice_id", id), zap.String("operation", "SendInvoice"))
logger.Info("sending invoice")

return s.repo.UpdateStatus(ctx, id, models.InvoiceStatusSent)
}

// MarkInvoicePaid отмечает счет как оплаченный
func (s *InvoiceService) MarkInvoicePaid(ctx context.Context, id string) error {
logger := loggerWith(ctx, s.log, zap.String("invoice_id", id), zap.String("operation", "MarkInvoicePaid"))
logger.Info("marking invoice as paid")

return s.repo.UpdateStatus(ctx, id, models.InvoiceStatusPaid)
}

// CancelInvoice отменяет счет
func (s *InvoiceService) CancelInvoice(ctx context.Context, id string) error {
logger := loggerWith(ctx, s.log, zap.String("invoice_id", id), zap.String("operation", "CancelInvoice"))
logger.Info("cancelling invoice")

return s.repo.UpdateStatus(ctx, id, models.InvoiceStatusCancelled)
}

// PaymentService сервис для работы с платежами
type PaymentService struct {
repo        PaymentRepoInterface
invoiceRepo InvoiceRepoInterface
log         *zap.Logger
}

type PaymentRepoInterface interface {
Create(ctx context.Context, payment models.Payment) error
GetByID(ctx context.Context, id string) (models.Payment, error)
GetByInvoiceID(ctx context.Context, invoiceID string) ([]models.Payment, error)
UpdateStatus(ctx context.Context, id, status string) error
}

func NewPaymentService(repo PaymentRepoInterface, invoiceRepo InvoiceRepoInterface, log *zap.Logger) *PaymentService {
return &PaymentService{
repo:        repo,
invoiceRepo: invoiceRepo,
log:         log,
}
}

// CreatePayment создает новый платеж
func (s *PaymentService) CreatePayment(ctx context.Context, req models.CreatePaymentRequest) (models.Payment, error) {
logger := loggerWith(ctx, s.log, zap.String("operation", "CreatePayment"))
logger.Info("creating payment")

paymentID := uuid.New().String()

payment := models.Payment{
ID:            paymentID,
InvoiceID:     req.InvoiceID,
Amount:        req.Amount,
PaymentMethod: req.PaymentMethod,
TransactionID: req.TransactionID,
PaymentDate:   time.Now(),
Metadata:      req.Metadata,
Status:        "pending",
CreatedAt:     time.Now(),
}

if err := s.repo.Create(ctx, payment); err != nil {
logger.Error("failed to create payment", zap.Error(err))
return models.Payment{}, err
}

// Проверяем, оплачен ли счет полностью
if err := s.checkInvoicePaid(ctx, req.InvoiceID); err != nil {
logger.Warn("failed to check invoice paid status", zap.Error(err))
}

logger.Info("payment created successfully", zap.String("payment_id", paymentID))
return payment, nil
}

// checkInvoicePaid проверяет, оплачен ли счет полностью
func (s *PaymentService) checkInvoicePaid(ctx context.Context, invoiceID string) error {
invoice, err := s.invoiceRepo.GetByID(ctx, invoiceID)
if err != nil {
return err
}

payments, err := s.repo.GetByInvoiceID(ctx, invoiceID)
if err != nil {
return err
}

var totalPaid float64
for _, p := range payments {
if p.Status == "completed" {
totalPaid += p.Amount
}
}

if totalPaid >= invoice.Total && invoice.Status != models.InvoiceStatusPaid {
return s.invoiceRepo.UpdateStatus(ctx, invoiceID, models.InvoiceStatusPaid)
}

return nil
}

// GetPayment получает платеж по ID
func (s *PaymentService) GetPayment(ctx context.Context, id string) (models.Payment, error) {
logger := loggerWith(ctx, s.log, zap.String("payment_id", id), zap.String("operation", "GetPayment"))
logger.Debug("fetching payment")

return s.repo.GetByID(ctx, id)
}

// GetPaymentsByInvoice получает платежи по счету
func (s *PaymentService) GetPaymentsByInvoice(ctx context.Context, invoiceID string) ([]models.Payment, error) {
logger := loggerWith(ctx, s.log, zap.String("invoice_id", invoiceID), zap.String("operation", "GetPaymentsByInvoice"))
logger.Debug("fetching payments")

return s.repo.GetByInvoiceID(ctx, invoiceID)
}

// ConfirmPayment подтверждает платеж
func (s *PaymentService) ConfirmPayment(ctx context.Context, id string) error {
logger := loggerWith(ctx, s.log, zap.String("payment_id", id), zap.String("operation", "ConfirmPayment"))
logger.Info("confirming payment")

payment, err := s.repo.GetByID(ctx, id)
if err != nil {
return err
}

if err := s.repo.UpdateStatus(ctx, id, "completed"); err != nil {
return err
}

// Проверяем, оплачен ли счет полностью
if err := s.checkInvoicePaid(ctx, payment.InvoiceID); err != nil {
logger.Warn("failed to check invoice paid status", zap.Error(err))
}

logger.Info("payment confirmed successfully")
return nil
}

// PaymentGatewayService сервис для работы с платежными шлюзами
type PaymentGatewayService struct {
repo PaymentGatewayRepoInterface
log  *zap.Logger
}

type PaymentGatewayRepoInterface interface {
GetAll(ctx context.Context) ([]models.PaymentGateway, error)
GetActive(ctx context.Context, provider string) (models.PaymentGateway, error)
Update(ctx context.Context, gw models.PaymentGateway) error
}

func NewPaymentGatewayService(repo PaymentGatewayRepoInterface, log *zap.Logger) *PaymentGatewayService {
return &PaymentGatewayService{
repo: repo,
log:  log,
}
}

// GetPaymentGateways получает все платежные шлюзы
func (s *PaymentGatewayService) GetPaymentGateways(ctx context.Context) ([]models.PaymentGateway, error) {
logger := loggerWith(ctx, s.log, zap.String("operation", "GetPaymentGateways"))
logger.Debug("fetching payment gateways")

return s.repo.GetAll(ctx)
}

// GetActivePaymentGateway получает активный шлюз по провайдеру
func (s *PaymentGatewayService) GetActivePaymentGateway(ctx context.Context, provider string) (models.PaymentGateway, error) {
logger := loggerWith(ctx, s.log, zap.String("provider", provider), zap.String("operation", "GetActivePaymentGateway"))
logger.Debug("fetching active payment gateway")

return s.repo.GetActive(ctx, provider)
}

// UpdatePaymentGateway обновляет конфигурацию шлюза
func (s *PaymentGatewayService) UpdatePaymentGateway(ctx context.Context, gw models.PaymentGateway) error {
logger := loggerWith(ctx, s.log, zap.String("gateway_id", gw.ID), zap.String("operation", "UpdatePaymentGateway"))
logger.Info("updating payment gateway")

gw.UpdatedAt = time.Now()
return s.repo.Update(ctx, gw)
}
