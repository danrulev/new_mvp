package handler

import (
	"desktop_lab/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) initMaterialRoutes(api *gin.RouterGroup) {
	// Материалы требуют аутентификации
	auth := api.Group("/materials")
	auth.Use(h.authMiddleware)
	{
		// Чтение материалов - все аутентифицированные
		auth.GET("/", h.permissionMiddleware(models.PermMaterialRead), h.getMaterialList)
		auth.GET("/:id", h.permissionMiddleware(models.PermMaterialRead), h.getMaterialByID)
		auth.GET("/:id/dimensions", h.permissionMiddleware(models.PermDimensionRead), h.getDimensions)
	}
}

func (h *Handler) getMaterialList(c *gin.Context) {
	materials, err := h.material.GetAll(c.Request.Context())
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getMaterialList", "service error", err)
		return
	}

	c.JSON(http.StatusOK, materials)
}

func (h *Handler) getMaterialByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getMaterialByID", "id param is empty", nil)
		return
	}

	material, err := h.material.GetByID(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getMaterialList", "service error", err)
		return
	}

	c.JSON(http.StatusOK, material)
}

func (h *Handler) getDimensions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getDimensions", "id param is empty", nil)
		return
	}

	dim, err := h.material.GetContextDimensionsByMaterialID(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getDimensions", "service error", err)
		return
	}

	c.JSON(http.StatusOK, dim)
}
