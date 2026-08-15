package handler

import (
	"desktop_lab/internal/models"
	"desktop_lab/pkg/valid"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// initOrderRoutes инициализирует маршруты для работы с заявками
func (h *Handler) initOrderRoutes(api *gin.RouterGroup) {
	h.log.Debug("Init order routes")

	orders := api.Group("/orders")
	orders.Use(h.authMiddleware)
	{
		// Создание заявки - доступно клиенту, менеджеру, админу
		orders.POST("/", h.permissionMiddleware(models.PermOrderCreate), h.createOrder)

		// Получение списка заявок - с фильтрацией по роли
		orders.GET("/", h.permissionMiddleware(models.PermOrderRead), h.listOrders)

		// Получение заявки по ID
		orders.GET("/:id", h.permissionMiddleware(models.PermOrderRead), h.getOrderByID)

		// Обновление заявки (только черновика)
		orders.PUT("/:id", h.permissionMiddleware(models.PermOrderUpdate), h.updateOrder)

		// Удаление заявки (мягкое удаление)
		orders.DELETE("/:id", h.permissionMiddleware(models.PermOrderDelete), h.deleteOrder)

		// Изменение статуса заявки
		orders.PUT("/:id/status", h.permissionMiddleware(models.PermOrderUpdate), h.changeOrderStatus)

		// Принятие заявки (только менеджер/админ)
		orders.POST("/:id/accept", h.permissionMiddleware(models.PermOrderAccept), h.acceptOrder)

		// Отклонение заявки (только менеджер/админ)
		orders.POST("/:id/reject", h.permissionMiddleware(models.PermOrderReject), h.rejectOrder)

		// Завершение заявки (только инженер/менеджер/админ)
		orders.POST("/:id/complete", h.permissionMiddleware(models.PermOrderComplete), h.completeOrder)

		// Позиции заявки
		orders.GET("/:id/items", h.permissionMiddleware(models.PermOrderRead), h.listOrderItems)
		orders.POST("/:id/items", h.permissionMiddleware(models.PermOrderUpdate), h.createOrderItem)
		orders.GET("/:id/items/:itemID", h.permissionMiddleware(models.PermOrderRead), h.getOrderItem)
		orders.PUT("/:id/items/:itemID", h.permissionMiddleware(models.PermOrderUpdate), h.updateOrderItem)
		orders.DELETE("/:id/items/:itemID", h.permissionMiddleware(models.PermOrderDelete), h.deleteOrderItem)

		// История workflow
		orders.GET("/:id/workflow", h.permissionMiddleware(models.PermOrderRead), h.getOrderWorkflow)
	}
}

// createOrder создает новую заявку
func (h *Handler) createOrder(c *gin.Context) {
	var req models.CreateOrderRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create order", "invalid request body", err)
		return
	}

	if err := valid.ValidateStruct(req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create order", "validation failed", err)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "create order", "user not authenticated", err)
		return
	}
	
	order, err := h.order.CreateOrder(c.Request.Context(), req, userID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "create order", "failed to create order", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":    order.ID,
		"order": order,
	})
}

// getOrderByID получает заявку по ID
func (h *Handler) getOrderByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get order by id", "id param is empty", nil)
		return
	}

	response, err := h.order.GetOrder(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get order by id", "failed to get order", err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// listOrders получает список заявок с фильтрацией
func (h *Handler) listOrders(c *gin.Context) {
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

	filter := models.OrderListFilter{
		Paginated: models.Paginated{
			Limit:  limit,
			Offset: offset,
		},
		OrganizationID: c.Query("organization_id"),
		CustomerID:     c.Query("customer_id"),
		Status:         models.OrderStatus(c.Query("status")),
		DateFrom:       c.Query("date_from"),
		DateTo:         c.Query("date_to"),
	}

	// Если пользователь не админ, фильтруем по его данным
	userID, _ := getUserIDFromContext(c)
	role, _ := getRoleFromContext(c)

	if role == models.RoleClient {
		// Клиент видит только свои заявки
		filter.CustomerID = userID
	} else if role == models.RoleTechnician {
		// Техник видит только заявки своей организации
		// Нужно получить organization_id из профиля пользователя
		// Пока оставляем пустым, фильтрация будет на уровне сервиса
	}

	data, err := h.order.ListOrders(c.Request.Context(), filter)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "list orders", "failed to list orders", err)
		return
	}

	c.JSON(http.StatusOK, data)
}

// updateOrder обновляет заявку
func (h *Handler) updateOrder(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "update order", "id param is empty", nil)
		return
	}

	var req models.UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "update order", "invalid request body", err)
		return
	}

	order, err := h.order.UpdateOrder(c.Request.Context(), id, req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "update order", "failed to update order", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"order": order})
}

// deleteOrder удаляет заявку (мягкое удаление)
func (h *Handler) deleteOrder(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "delete order", "id param is empty", nil)
		return
	}

	if err := h.order.DeleteOrder(c.Request.Context(), id); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "delete order", "failed to delete order", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order deleted"})
}

// changeOrderStatus изменяет статус заявки
func (h *Handler) changeOrderStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "change order status", "id param is empty", nil)
		return
	}

	var req struct {
		Status  models.OrderStatus `json:"status" validate:"required"`
		Comment string             `json:"comment"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "change order status", "invalid request body", err)
		return
	}

	if err := valid.ValidateStruct(req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "change order status", "validation failed", err)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "change order status", "user not authenticated", err)
		return
	}
	
	// Получаем имя пользователя из контекста или профиля
	userName := "User"
	if nameRaw, exists := c.Get("user_name"); exists {
		if name, ok := nameRaw.(string); ok && name != "" {
			userName = name
		}
	}
	
	err = h.order.ChangeOrderStatus(c.Request.Context(), id, string(req.Status), userID, userName, req.Comment)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "change order status", "failed to change status", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status changed", "status": req.Status})
}

// acceptOrder принимает заявку
func (h *Handler) acceptOrder(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "accept order", "id param is empty", nil)
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "accept order", "invalid request body", err)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "accept order", "user not authenticated", err)
		return
	}
	
	// Получаем имя пользователя из контекста или профиля
	userName := "User"
	if nameRaw, exists := c.Get("user_name"); exists {
		if name, ok := nameRaw.(string); ok && name != "" {
			userName = name
		}
	}
	
	if err := h.order.AcceptOrder(c.Request.Context(), id, userID, userName, req.Comment); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "accept order", "failed to accept order", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order accepted"})
}

// rejectOrder отклоняет заявку
func (h *Handler) rejectOrder(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "reject order", "id param is empty", nil)
		return
	}

	var req struct {
		Comment string `json:"comment" validate:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "reject order", "invalid request body", err)
		return
	}

	if err := valid.ValidateStruct(req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "reject order", "comment is required", err)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "reject order", "user not authenticated", err)
		return
	}
	
	// Получаем имя пользователя из контекста или профиля
	userName := "User"
	if nameRaw, exists := c.Get("user_name"); exists {
		if name, ok := nameRaw.(string); ok && name != "" {
			userName = name
		}
	}
	
	if err := h.order.RejectOrder(c.Request.Context(), id, userID, userName, req.Comment); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "reject order", "failed to reject order", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order rejected"})
}

// completeOrder завершает заявку
func (h *Handler) completeOrder(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "complete order", "id param is empty", nil)
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "complete order", "invalid request body", err)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "complete order", "user not authenticated", err)
		return
	}
	
	// Получаем имя пользователя из контекста или профиля
	userName := "User"
	if nameRaw, exists := c.Get("user_name"); exists {
		if name, ok := nameRaw.(string); ok && name != "" {
			userName = name
		}
	}
	
	if err := h.order.CompleteOrder(c.Request.Context(), id, userID, userName, req.Comment); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "complete order", "failed to complete order", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order completed"})
}

// listOrderItems получает список позиций заявки
func (h *Handler) listOrderItems(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "list order items", "order id param is empty", nil)
		return
	}

	items, err := h.order.GetOrderItems(c.Request.Context(), orderID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "list order items", "failed to list items", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"meta": models.PaginatedMetadata{
			Total: int64(len(items)),
		},
	})
}

// createOrderItem создает позицию в заявке
func (h *Handler) createOrderItem(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "create order item", "order id param is empty", nil)
		return
	}

	var req models.CreateOrderItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create order item", "invalid request body", err)
		return
	}

	if err := valid.ValidateStruct(req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create order item", "validation failed", err)
		return
	}

	item, err := h.order.CreateOrderItem(c.Request.Context(), orderID, req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "create order item", "failed to create item", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"item": item})
}

// getOrderItem получает позицию заявки по ID
func (h *Handler) getOrderItem(c *gin.Context) {
	itemID := c.Param("itemID")
	if itemID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get order item", "item id param is empty", nil)
		return
	}

	item, err := h.order.GetOrderItem(c.Request.Context(), itemID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get order item", "failed to get item", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"item": item})
}

// updateOrderItem обновляет позицию заявки
func (h *Handler) updateOrderItem(c *gin.Context) {
	itemID := c.Param("itemID")
	if itemID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "update order item", "item id param is empty", nil)
		return
	}

	var req models.UpdateOrderItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "update order item", "invalid request body", err)
		return
	}

	item, err := h.order.UpdateOrderItem(c.Request.Context(), itemID, req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "update order item", "failed to update item", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"item": item})
}

// deleteOrderItem удаляет позицию заявки
func (h *Handler) deleteOrderItem(c *gin.Context) {
	itemID := c.Param("itemID")
	if itemID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "delete order item", "item id param is empty", nil)
		return
	}

	if err := h.order.DeleteOrderItem(c.Request.Context(), itemID); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "delete order item", "failed to delete item", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "item deleted"})
}

// getOrderWorkflow получает историю изменения статусов заявки
func (h *Handler) getOrderWorkflow(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get order workflow", "order id param is empty", nil)
		return
	}

	workflow, err := h.order.GetOrderWorkflow(c.Request.Context(), orderID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get order workflow", "failed to get workflow", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"workflow": workflow})
}
