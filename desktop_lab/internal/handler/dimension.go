package handler

import (
	"desktop_lab/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) initDimensionRoutes(api *gin.RouterGroup) {
	group := api.Group("/dimension")
	{
		group.GET("/", h.getAvailableDimensions)
		group.POST("/", h.addDimension)
		group.GET("/key/:key", h.getDimensionByKey)
		group.PUT("/:id/values", h.updatePossibleValues)
		group.DELETE("/:id/values", h.deletePossibleValues)
		group.DELETE("/:id", h.deleteDimension)
	}
}

func (h *Handler) getAvailableDimensions(c *gin.Context) {
	dims, err := h.dimension.GetAvailableDimensions(c.Request.Context())
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, dims)
}

func (h *Handler) addDimension(c *gin.Context) {
	var dim models.ContextDimension
	if err := c.BindJSON(&dim); err != nil {
		newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.dimension.AddDimension(c.Request.Context(), dim); err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "created"})
}

func (h *Handler) getDimensionByKey(c *gin.Context) {
	key := c.Param("key")
	dim, err := h.dimension.GetDimensionByKey(c.Request.Context(), key)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, dim)
}

func (h *Handler) updatePossibleValues(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Values []string `json:"values" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.dimension.UpdatePossibleValues(c.Request.Context(), id, req.Values); err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *Handler) deletePossibleValues(c *gin.Context) {
	id := c.Param("id")
	if err := h.dimension.DeletePossibleValues(c.Request.Context(), id); err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "values_deleted"})
}

func (h *Handler) deleteDimension(c *gin.Context) {
	id := c.Param("id")
	if err := h.dimension.DeleteDimension(c.Request.Context(), id); err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
