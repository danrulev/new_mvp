package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) initMaterialRoutes(api *gin.RouterGroup) {
	material := api.Group("/materials")
	{
		material.GET("/", h.getMaterialList)
		material.GET("/:id", h.getMaterialByID)
		material.GET("/:id/dimensions", h.getDimensions)
	}
}

func (h *Handler) getMaterialList(c *gin.Context) {
	materials, err := h.material.GetAll(c.Request.Context())
	if err != nil {
		newErrorResponse(c, 500, err.Error())
		return
	}

	c.JSON(http.StatusOK, materials)
}

func (h *Handler) getMaterialByID(c *gin.Context) {
	id := c.Param("id")

	material, err := h.material.GetByID(c.Request.Context(), id)
	if err != nil {
		newErrorResponse(c, 500, err.Error())
		return
	}

	c.JSON(http.StatusOK, material)
}

func (h *Handler) getDimensions(c *gin.Context) {
	id := c.Param("id")

	dim, err := h.material.GetContextDimensionsByMaterialID(c.Request.Context(), id)
	if err != nil {
		newErrorResponse(c, 500, err.Error())
		return
	}

	c.JSON(http.StatusOK, dim)
}
