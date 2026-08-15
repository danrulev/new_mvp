package handler

import (
"desktop_lab/internal/models"
"desktop_lab/internal/service"
"desktop_lab/pkg/valid"
"net/http"
"strconv"

"github.com/gin-gonic/gin"
)

// initInvitationRoutes инициализирует маршруты для работы с приглашениями
func (h *Handler) initInvitationRoutes(api *gin.RouterGroup) {
h.log.Debug("Init invitation routes")

invitations := api.Group("/invitations")
invitations.Use(h.authMiddleware)
{
invitations.POST("/", h.permissionMiddleware(models.PermInvitationCreate), h.createInvitation)
invitations.GET("/", h.permissionMiddleware(models.PermInvitationRead), h.listInvitations)
invitations.GET("/:id", h.permissionMiddleware(models.PermInvitationRead), h.getInvitationByID)
invitations.POST("/accept", h.acceptInvitation)
invitations.POST("/decline", h.declineInvitation)
invitations.DELETE("/:id", h.permissionMiddleware(models.PermInvitationCreate), h.revokeInvitation)
invitations.POST("/:id/resend", h.permissionMiddleware(models.PermInvitationCreate), h.resendInvitation)
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
userID, err := getUserIDFromContext(c)
if err != nil {
h.newErrorResponse(c, http.StatusUnauthorized, "create invitation", "user not authenticated", err)
return
}
userName := "User"
if nameRaw, exists := c.Get("user_name"); exists {
if name, ok := nameRaw.(string); ok && name != "" {
userName = name
}
}
invitation, err := h.invitation.CreateInvitation(c.Request.Context(), req, userID, userName)
if err != nil {
h.newErrorResponse(c, http.StatusInternalServerError, "create invitation", "failed to create invitation", err)
return
}
c.JSON(http.StatusCreated, gin.H{"id": invitation.ID, "invitation": invitation})
}

func (h *Handler) getInvitationByID(c *gin.Context) {
id := c.Param("id")
if id == "" {
h.newErrorResponse(c, http.StatusBadRequest, "get invitation by id", "id param is empty", nil)
return
}
invitation, err := h.invitation.GetInvitation(c.Request.Context(), id)
if err != nil {
h.newErrorResponse(c, http.StatusInternalServerError, "get invitation by id", "failed to get invitation", err)
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
Paginated:      models.Paginated{Limit: limit, Offset: offset},
OrganizationID: c.Query("organization_id"),
Email:          c.Query("email"),
Status:         models.InvitationStatus(c.Query("status")),
InvitedBy:      c.Query("invited_by"),
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

func (h *Handler) acceptInvitation(c *gin.Context) {
var req models.AcceptInvitationRequest
if err := c.ShouldBindJSON(&req); err != nil {
h.newErrorResponse(c, http.StatusBadRequest, "accept invitation", "invalid request body", err)
return
}
if err := valid.ValidateStruct(req); err != nil {
h.newErrorResponse(c, http.StatusBadRequest, "accept invitation", "validation failed", err)
return
}
userID, err := getUserIDFromContext(c)
if err != nil {
h.newErrorResponse(c, http.StatusUnauthorized, "accept invitation", "user not authenticated", err)
return
}
if err := h.invitation.AcceptInvitation(c.Request.Context(), req.Token, userID); err != nil {
switch err {
case service.ErrInvitationNotFound:
h.newErrorResponse(c, http.StatusNotFound, "accept invitation", "invitation not found", err)
case service.ErrInvitationExpired:
h.newErrorResponse(c, http.StatusBadRequest, "accept invitation", "invitation expired", err)
case service.ErrInvitationAlreadyUsed:
h.newErrorResponse(c, http.StatusBadRequest, "accept invitation", "invitation already used", err)
default:
h.newErrorResponse(c, http.StatusInternalServerError, "accept invitation", "failed to accept invitation", err)
}
return
}
c.JSON(http.StatusOK, gin.H{"message": "invitation accepted"})
}

func (h *Handler) declineInvitation(c *gin.Context) {
var req struct {
Token string `json:"token" validate:"required"`
}
if err := c.ShouldBindJSON(&req); err != nil {
h.newErrorResponse(c, http.StatusBadRequest, "decline invitation", "invalid request body", err)
return
}
if err := valid.ValidateStruct(req); err != nil {
h.newErrorResponse(c, http.StatusBadRequest, "decline invitation", "validation failed", err)
return
}
if err := h.invitation.DeclineInvitation(c.Request.Context(), req.Token); err != nil {
if err == service.ErrInvitationNotFound {
h.newErrorResponse(c, http.StatusNotFound, "decline invitation", "invitation not found", err)
} else {
h.newErrorResponse(c, http.StatusInternalServerError, "decline invitation", "failed to decline invitation", err)
}
return
}
c.JSON(http.StatusOK, gin.H{"message": "invitation declined"})
}

func (h *Handler) revokeInvitation(c *gin.Context) {
id := c.Param("id")
if id == "" {
h.newErrorResponse(c, http.StatusBadRequest, "revoke invitation", "id param is empty", nil)
return
}
if err := h.invitation.RevokeInvitation(c.Request.Context(), id); err != nil {
if err == service.ErrInvitationNotFound {
h.newErrorResponse(c, http.StatusNotFound, "revoke invitation", "invitation not found", err)
} else {
h.newErrorResponse(c, http.StatusInternalServerError, "revoke invitation", "failed to revoke invitation", err)
}
return
}
c.JSON(http.StatusOK, gin.H{"message": "invitation revoked"})
}

func (h *Handler) resendInvitation(c *gin.Context) {
id := c.Param("id")
if id == "" {
h.newErrorResponse(c, http.StatusBadRequest, "resend invitation", "id param is empty", nil)
return
}
invitation, err := h.invitation.ResendInvitation(c.Request.Context(), id)
if err != nil {
if err == service.ErrInvitationNotFound {
h.newErrorResponse(c, http.StatusNotFound, "resend invitation", "invitation not found", err)
} else {
h.newErrorResponse(c, http.StatusInternalServerError, "resend invitation", "failed to resend invitation", err)
}
return
}
c.JSON(http.StatusOK, gin.H{"message": "invitation resent", "invitation": invitation})
}
