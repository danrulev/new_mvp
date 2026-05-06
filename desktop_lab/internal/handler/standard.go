package handler

import (
	"desktop_lab/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) initStandardRoutes(api *gin.RouterGroup) {
	group := api.Group("/standard")
	{
		group.POST("/", h.createStandard)
		group.GET("/material/:id", h.getStandardsByMaterialID)
		group.GET("/:id/methods", h.getMethodsByStandardID)
		group.GET("/:id/methods/full", h.getMethodsFullByStandardID)
		group.GET("/method/:id/details", h.getMethodDetails)
		group.GET("/:id/dimensions", h.getStandardDimensions)
		group.GET("/:id/full", h.getStandardFull)
		group.DELETE("/cache/:id", h.invalidateStandardCache)
		group.POST("/:id/dimensions/:dimId", h.linkDimensionToStandard)
	}
}

func (h *Handler) createStandard(c *gin.Context) {
	var req models.CreateStandardRequest
	if err := c.BindJSON(&req); err != nil {
		newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	id, err := h.standard.CreateStandard(c.Request.Context(), req)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handler) getStandardsByMaterialID(c *gin.Context) {
	matID := c.Param("id")
	stds, err := h.standard.GetByMaterialID(c.Request.Context(), matID)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, stds)
}

func (h *Handler) getMethodsByStandardID(c *gin.Context) {
	stdID := c.Param("id")
	methods, err := h.standard.GetMethodsByStandardID(c.Request.Context(), stdID)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, methods)
}

func (h *Handler) getMethodsFullByStandardID(c *gin.Context) {
	stdID := c.Param("id")
	methods, err := h.standard.GetMethodsFullByStandardIDWithCache(c.Request.Context(), stdID)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, methods)
}

func (h *Handler) getMethodDetails(c *gin.Context) {
	methodID := c.Param("id")
	details, err := h.standard.GetMethodDetails(c.Request.Context(), methodID)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, details)
}

func (h *Handler) getStandardDimensions(c *gin.Context) {
	stdID := c.Param("id")
	dims, err := h.standard.GetStandardDimensions(c.Request.Context(), stdID)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, dims)
}

func (h *Handler) getStandardFull(c *gin.Context) {
	stdID := c.Param("id")
	full, err := h.standard.GetStandardFull(c.Request.Context(), stdID)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, full)
}

func (h *Handler) invalidateStandardCache(c *gin.Context) {
	stdID := c.Param("id")
	h.standard.InvalidateStandardCache(stdID)
	c.JSON(http.StatusOK, gin.H{"status": "cache_invalidated"})
}

func (h *Handler) linkDimensionToStandard(c *gin.Context) {
	stdID := c.Param("id")
	dimID := c.Param("dimId")
	if err := h.standard.LinkDimensionToStandard(c.Request.Context(), stdID, dimID); err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "linked"})
}
