package handler

import (
	"desktop_lab/internal/models"
	"desktop_lab/pkg/valid"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// initFinanceRoutes инициализирует маршруты для финансового модуля
func (h *Handler) initFinanceRoutes(api *gin.RouterGroup) {
	h.log.Debug("Init finance routes")

	// Прайс-листы
	priceLists := api.Group("/price-lists")
	priceLists.Use(h.authMiddleware)
	{
		priceLists.POST("/", h.permissionMiddleware(models.PermAdmin), h.createPriceList)
		priceLists.GET("/", h.permissionMiddleware(models.PermOrderRead), h.listPriceLists)
		priceLists.GET("/:id", h.permissionMiddleware(models.PermOrderRead), h.getPriceList)
		priceLists.GET("/:id/items", h.permissionMiddleware(models.PermOrderRead), h.getPriceListItems)
		priceLists.POST("/:id/items", h.permissionMiddleware(models.PermAdmin), h.addPriceListItem)
	}

	// Счета
	invoices := api.Group("/invoices")
	invoices.Use(h.authMiddleware)
	{
		invoices.POST("/", h.permissionMiddleware(models.PermOrderCreate), h.createInvoice)
		invoices.GET("/", h.permissionMiddleware(models.PermOrderRead), h.listInvoices)
		invoices.GET("/:id", h.permissionMiddleware(models.PermOrderRead), h.getInvoice)
		invoices.POST("/:id/send", h.permissionMiddleware(models.PermOrderUpdate), h.sendInvoice)
		invoices.POST("/:id/mark-paid", h.permissionMiddleware(models.PermAdmin), h.markInvoicePaid)
		invoices.POST("/:id/cancel", h.permissionMiddleware(models.PermOrderUpdate), h.cancelInvoice)
	}

	// Платежи
	payments := api.Group("/payments")
	payments.Use(h.authMiddleware)
	{
		payments.POST("/", h.permissionMiddleware(models.PermOrderCreate), h.createPayment)
		payments.GET("/:id", h.permissionMiddleware(models.PermOrderRead), h.getPayment)
		payments.GET("/invoice/:invoiceID", h.permissionMiddleware(models.PermOrderRead), h.getPaymentsByInvoice)
		payments.POST("/:id/confirm", h.permissionMiddleware(models.PermAdmin), h.confirmPayment)
	}

	// Платежные шлюзы
	gateways := api.Group("/payment-gateways")
	gateways.Use(h.authMiddleware)
	{
		gateways.GET("/", h.permissionMiddleware(models.PermAdmin), h.getPaymentGateways)
		gateways.GET("/:provider", h.permissionMiddleware(models.PermAdmin), h.getActivePaymentGateway)
		gateways.PUT("/:id", h.permissionMiddleware(models.PermAdmin), h.updatePaymentGateway)
	}
}

// createPriceList создает новый прайс-лист
func (h *Handler) createPriceList(c *gin.Context) {
	var req models.CreatePriceListRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create price list", "invalid request body", err)
		return
	}

	if err := valid.ValidateStruct(req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create price list", "validation failed", err)
		return
	}

	priceList, err := h.priceList.CreatePriceList(c.Request.Context(), req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "create price list", "failed to create price list", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"price_list": priceList})
}

// getPriceList получает прайс-лист по ID
func (h *Handler) getPriceList(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get price list", "id param is empty", nil)
		return
	}

	priceList, err := h.priceList.GetPriceList(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get price list", "failed to get price list", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"price_list": priceList})
}

// listPriceLists получает список прайс-листов
func (h *Handler) listPriceLists(c *gin.Context) {
	orgID := c.Query("organization_id")
	if orgID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "list price lists", "organization_id is required", nil)
		return
	}

	priceLists, err := h.priceList.GetPriceListsByOrganization(c.Request.Context(), orgID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "list price lists", "failed to get price lists", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"price_lists": priceLists,
		"meta": models.PaginatedMetadata{
			Total: int64(len(priceLists)),
		},
	})
}

// addPriceListItem добавляет позицию в прайс-лист
func (h *Handler) addPriceListItem(c *gin.Context) {
	var req models.CreatePriceListItemRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "add price list item", "invalid request body", err)
		return
	}

	if err := valid.ValidateStruct(req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "add price list item", "validation failed", err)
		return
	}

	item, err := h.priceList.AddPriceListItem(c.Request.Context(), req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "add price list item", "failed to add item", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"item": item})
}

// getPriceListItems получает позиции прайс-листа
func (h *Handler) getPriceListItems(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get price list items", "id param is empty", nil)
		return
	}

	items, err := h.priceList.GetPriceListItems(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get price list items", "failed to get items", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"meta": models.PaginatedMetadata{
			Total: int64(len(items)),
		},
	})
}

// createInvoice создает новый счет
func (h *Handler) createInvoice(c *gin.Context) {
	var req models.CreateInvoiceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create invoice", "invalid request body", err)
		return
	}

	if err := valid.ValidateStruct(req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create invoice", "validation failed", err)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "create invoice", "user not authenticated", err)
		return
	}

	invoice, err := h.invoice.CreateInvoice(c.Request.Context(), req, userID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "create invoice", "failed to create invoice", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"invoice": invoice})
}

// getInvoice получает счет по ID
func (h *Handler) getInvoice(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get invoice", "id param is empty", nil)
		return
	}

	response, err := h.invoice.GetInvoice(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get invoice", "failed to get invoice", err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// listInvoices получает список счетов
func (h *Handler) listInvoices(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 {
		limit = 10
	}

	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil || offset < 0 {
		offset = 0
	}

	filter := models.InvoiceListFilter{
		Paginated: models.Paginated{
			Limit:  limit,
			Offset: offset,
		},
		OrderID:    c.Query("order_id"),
		CustomerID: c.Query("customer_id"),
		Status:     models.InvoiceStatus(c.Query("status")),
		DateFrom:   c.Query("date_from"),
		DateTo:     c.Query("date_to"),
	}

	data, err := h.invoice.ListInvoices(c.Request.Context(), filter)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "list invoices", "failed to list invoices", err)
		return
	}

	c.JSON(http.StatusOK, data)
}

// sendInvoice отправляет счет
func (h *Handler) sendInvoice(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "send invoice", "id param is empty", nil)
		return
	}

	if err := h.invoice.SendInvoice(c.Request.Context(), id); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "send invoice", "failed to send invoice", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "invoice sent"})
}

// markInvoicePaid отмечает счет как оплаченный
func (h *Handler) markInvoicePaid(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "mark invoice paid", "id param is empty", nil)
		return
	}

	if err := h.invoice.MarkInvoicePaid(c.Request.Context(), id); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "mark invoice paid", "failed to mark as paid", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "invoice marked as paid"})
}

// cancelInvoice отменяет счет
func (h *Handler) cancelInvoice(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "cancel invoice", "id param is empty", nil)
		return
	}

	if err := h.invoice.CancelInvoice(c.Request.Context(), id); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "cancel invoice", "failed to cancel invoice", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "invoice cancelled"})
}

// createPayment создает новый платеж
func (h *Handler) createPayment(c *gin.Context) {
	var req models.CreatePaymentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create payment", "invalid request body", err)
		return
	}

	if err := valid.ValidateStruct(req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create payment", "validation failed", err)
		return
	}

	payment, err := h.payment.CreatePayment(c.Request.Context(), req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "create payment", "failed to create payment", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"payment": payment})
}

// getPayment получает платеж по ID
func (h *Handler) getPayment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get payment", "id param is empty", nil)
		return
	}

	payment, err := h.payment.GetPayment(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get payment", "failed to get payment", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"payment": payment})
}

// getPaymentsByInvoice получает платежи по счету
func (h *Handler) getPaymentsByInvoice(c *gin.Context) {
	invoiceID := c.Param("invoiceID")
	if invoiceID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get payments by invoice", "invoiceID param is empty", nil)
		return
	}

	payments, err := h.payment.GetPaymentsByInvoice(c.Request.Context(), invoiceID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get payments by invoice", "failed to get payments", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payments": payments,
		"meta": models.PaginatedMetadata{
			Total: int64(len(payments)),
		},
	})
}

// confirmPayment подтверждает платеж
func (h *Handler) confirmPayment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "confirm payment", "id param is empty", nil)
		return
	}

	if err := h.payment.ConfirmPayment(c.Request.Context(), id); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "confirm payment", "failed to confirm payment", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "payment confirmed"})
}

// getPaymentGateways получает все платежные шлюзы
func (h *Handler) getPaymentGateways(c *gin.Context) {
	gateways, err := h.paymentGateway.GetPaymentGateways(c.Request.Context())
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get payment gateways", "failed to get gateways", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"gateways": gateways})
}

// getActivePaymentGateway получает активный шлюз по провайдеру
func (h *Handler) getActivePaymentGateway(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get active payment gateway", "provider param is empty", nil)
		return
	}

	gateway, err := h.paymentGateway.GetActivePaymentGateway(c.Request.Context(), provider)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get active payment gateway", "failed to get gateway", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"gateway": gateway})
}

// updatePaymentGateway обновляет конфигурацию шлюза
func (h *Handler) updatePaymentGateway(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "update payment gateway", "id param is empty", nil)
		return
	}

	var gw models.PaymentGateway
	if err := c.ShouldBindJSON(&gw); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "update payment gateway", "invalid request body", err)
		return
	}

	gw.ID = id

	if err := h.paymentGateway.UpdatePaymentGateway(c.Request.Context(), gw); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "update payment gateway", "failed to update gateway", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "payment gateway updated"})
}
