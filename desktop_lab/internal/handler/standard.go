package handler

import (
	"desktop_lab/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) initStandardRoutes(api *gin.RouterGroup) {
	// Стандарты требуют аутентификации
	auth := api.Group("/standard")
	auth.Use(h.authMiddleware)
	{
		// Создание стандарта - только инженер и админ
		auth.POST("/", h.permissionMiddleware(models.PermStandardCreate), h.createStandard)
		// Чтение стандартов - все аутентифицированные
		auth.GET("/material/:id", h.permissionMiddleware(models.PermStandardRead), h.getStandardsByMaterialID)
		auth.GET("/:id/methods", h.permissionMiddleware(models.PermStandardRead), h.getMethodsByStandardID)
		auth.GET("/:id/methods/full", h.permissionMiddleware(models.PermStandardRead), h.getMethodsFullByStandardID)
		auth.GET("/method/:id/details", h.permissionMiddleware(models.PermStandardRead), h.getMethodDetails)
		auth.GET("/:id/dimensions", h.permissionMiddleware(models.PermStandardRead), h.getStandardDimensions)
		auth.GET("/:id/full", h.permissionMiddleware(models.PermStandardRead), h.getStandardFull)
		// Инвалидация кэша - только инженер и админ
		auth.DELETE("/cache/:id", h.permissionMiddleware(models.PermStandardUpdate), h.invalidateStandardCache)
		// Линковка измерений - только инженер и админ
		auth.POST("/:id/dimensions/:dimId", h.permissionMiddleware(models.PermStandardUpdate), h.linkDimensionToStandard)
	}
}

func (h *Handler) createStandard(c *gin.Context) {
	var req models.CreateStandardRequest
	if err := c.BindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "createStandard", "invalid data", err)
		return
	}

	id, err := h.standard.CreateStandard(c.Request.Context(), req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "createStandard", "service error", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handler) getStandardsByMaterialID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getStandardsByMaterialID", "id param is empty", nil)
		return
	}

	stds, err := h.standard.GetByMaterialID(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getStandardsByMaterialID", "service error", err)
		return
	}
	c.JSON(http.StatusOK, stds)
}

func (h *Handler) getMethodsByStandardID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getMethodsByStandardID", "id param is empty", nil)
		return
	}
	methods, err := h.standard.GetMethodsByStandardID(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getMethodsByStandardID", "service error", err)
		return
	}
	c.JSON(http.StatusOK, methods)
}

func (h *Handler) getMethodsFullByStandardID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getMethodsFullByStandardID", "id param is empty", nil)
		return
	}
	methods, err := h.standard.GetMethodsFullByStandardIDWithCache(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getMethodsFullByStandardID", "service error", err)
		return
	}
	c.JSON(http.StatusOK, methods)
}

func (h *Handler) getMethodDetails(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getMethodDetails", "id param is empty", nil)
		return
	}

	details, err := h.standard.GetMethodDetails(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getMethodDetails", "service error", err)
		return
	}
	c.JSON(http.StatusOK, details)
}

func (h *Handler) getStandardDimensions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getStandardDimensions", "id param is empty", nil)
		return
	}

	dims, err := h.standard.GetStandardDimensions(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getStandardDimensions", "service error", err)
		return
	}
	c.JSON(http.StatusOK, dims)
}

func (h *Handler) getStandardFull(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getStandardFull", "id param is empty", nil)
		return
	}

	full, err := h.standard.GetStandardFull(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getStandardFull", "service error", err)
		return
	}
	c.JSON(http.StatusOK, full)
}

func (h *Handler) invalidateStandardCache(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "invalidateStandardCache", "id param is empty", nil)
		return
	}

	h.standard.InvalidateStandardCache(id)
	c.JSON(http.StatusOK, gin.H{"status": "cache_invalidated"})
}

func (h *Handler) linkDimensionToStandard(c *gin.Context) {
	stdID := c.Param("stdID")
	if stdID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "linkDimensionToStandard", "stdID param is empty", nil)
		return
	}

	dimID := c.Param("dimID")
	if stdID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "linkDimensionToStandard", "dimID param is empty", nil)
		return
	}

	if err := h.standard.LinkDimensionToStandard(c.Request.Context(), stdID, dimID); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "linkDimensionToStandard", "id param is empty", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "linked"})
}
