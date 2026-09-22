package handler

import (
	"desktop_lab/internal/models"
	"desktop_lab/internal/service"
	"desktop_lab/pkg/valid"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// initInvitationRoutes инициализирует маршруты для работы с заявками на регистрацию
func (h *Handler) initInvitationRoutes(api *gin.RouterGroup) {
	h.log.Debug("Init invitation routes")

	invitations := api.Group("/invitations")
	{
		// Публичный маршрут для создания заявки на регистрацию
		invitations.POST("/", h.createInvitation)
		
		// Маршруты для администраторов
		invitations.Use(h.authMiddleware)
		invitations.GET("/", h.permissionMiddleware(models.PermInvitationRead), h.listInvitations)
		invitations.GET("/:id", h.permissionMiddleware(models.PermInvitationRead), h.getInvitationByID)
		invitations.POST("/:id/review", h.permissionMiddleware(models.PermUserCreate), h.reviewInvitation)
		invitations.DELETE("/:id", h.permissionMiddleware(models.PermUserCreate), h.deleteInvitation)
	}
}

// initQualityControlRoutes инициализирует маршруты контроля качества
func (h *Handler) initQualityControlRoutes(api *gin.RouterGroup) {
	h.log.Debug("Init quality control routes")

	handler := NewQualityControlHandler(h.qualityControl, h.log)
	quality := api.Group("/quality")
	quality.Use(h.authMiddleware)
	{
		handler.RegisterRoutes(quality)
	}
}

func (h *Handler) createInvitation(c *gin.Context) {
	var req models.CreateInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create invitation", "invalid request body", err)
		return
	}
	if err := valid.ValidateStruct(req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create invitation", "validation failed", err)
		return
	}

	invitation, err := h.invitation.CreateInvitation(c.Request.Context(), req)
	if err != nil {
		switch err {
		case service.ErrInvalidRole:
			h.newErrorResponse(c, http.StatusBadRequest, "create invitation", "invalid role", err)
		case service.ErrInvitationExists:
			h.newErrorResponse(c, http.StatusConflict, "create invitation", "invitation already exists", err)
		default:
			h.newErrorResponse(c, http.StatusInternalServerError, "create invitation", "failed to create invitation", err)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"invitation": invitation})
}

func (h *Handler) getInvitationByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get invitation by id", "id param is empty", nil)
		return
	}

	invitation, err := h.invitation.GetInvitation(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrInvitationNotFound {
			h.newErrorResponse(c, http.StatusNotFound, "get invitation by id", "invitation not found", err)
		} else {
			h.newErrorResponse(c, http.StatusInternalServerError, "get invitation by id", "failed to get invitation", err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"invitation": invitation})
}

func (h *Handler) listInvitations(c *gin.Context) {
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

	filter := models.InvitationListFilter{
		Paginated: models.Paginated{Limit: limit, Offset: offset},
		Email:     c.Query("email"),
		Status:    models.InvitationStatus(c.Query("status")),
	}

	invitations, total, err := h.invitation.ListInvitations(c.Request.Context(), filter)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "list invitations", "failed to list invitations", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invitations": invitations,
		"meta":        models.PaginatedMetadata{Total: total, Limit: limit, Offset: offset},
	})
}

func (h *Handler) reviewInvitation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "review invitation", "id param is empty", nil)
		return
	}

	var req models.ReviewInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "review invitation", "invalid request body", err)
		return
	}
	if err := valid.ValidateStruct(req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "review invitation", "validation failed", err)
		return
	}

	adminID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "review invitation", "user not authenticated", err)
		return
	}

	if err := h.invitation.ReviewInvitation(c.Request.Context(), id, adminID, req); err != nil {
		switch err {
		case service.ErrInvitationNotFound:
			h.newErrorResponse(c, http.StatusNotFound, "review invitation", "invitation not found", err)
		case service.ErrAlreadyReviewed:
			h.newErrorResponse(c, http.StatusBadRequest, "review invitation", "invitation already reviewed", err)
		default:
			h.newErrorResponse(c, http.StatusInternalServerError, "review invitation", "failed to review invitation", err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "invitation reviewed"})
}

func (h *Handler) deleteInvitation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "delete invitation", "id param is empty", nil)
		return
	}

	if err := h.invitation.DeleteInvitation(c.Request.Context(), id); err != nil {
		if err == service.ErrInvitationNotFound {
			h.newErrorResponse(c, http.StatusNotFound, "delete invitation", "invitation not found", err)
		} else {
			h.newErrorResponse(c, http.StatusInternalServerError, "delete invitation", "failed to delete invitation", err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "invitation deleted"})
}
