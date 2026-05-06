package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) initSampleRoutes(api *gin.RouterGroup) {
	group := api.Group("/sample")
	{
		group.POST("/", h.createSample)
	}
}

func (h *Handler) createSample(c *gin.Context) {
	var req struct {
		GroupID       string            `json:"group_id" binding:"required"`
		Number        string            `json:"number" binding:"required"`
		ContextParams map[string]string `json:"context_params"`
		Note          string            `json:"note"`
	}
	if err := c.BindJSON(&req); err != nil {
		newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	sample, err := h.sample.CreateSample(c.Request.Context(), req.GroupID, req.Number, req.ContextParams, req.Note)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, sample)
}