package handler

import (
	"desktop_lab/internal/models"
	"desktop_lab/pkg/valid"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) initProfileRoutes(api *gin.RouterGroup) {
	h.log.Debug("init profile routes")
	profile := api.Group("/profile")
	profile.Use(h.authMiddleware)
	{
		profile.GET("/", h.getProfile)
		profile.PUT("/", h.updateProfile)
		profile.DELETE("/", h.deleteProfile)
	}

	// Endpoint для получения списка активных сотрудников
	api.GET("/employees",
		h.authMiddleware,
		h.permissionMiddleware(models.PermUserRead),
		h.getEmployeesList,
	)
}

func (h *Handler) getProfile(c *gin.Context) {
	h.log.Debug("get profile")
	userID, err := getUserID(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "get profile", "unauthorized", err)
		return
	}

	profile, err := h.profile.GetProfile(c.Request.Context(), userID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get profile", "service error", err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *Handler) updateProfile(c *gin.Context) {
	h.log.Debug("update profile")

	userID, err := getUserID(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "update profile", "unauthorized", err)
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "update profile", "invalid data", err)
		return
	}

	if err := valid.ValidateStruct(req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "update profile", "invalid data", err)
		return
	}

	user, err := h.profile.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "update profile", "service error", err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) deleteProfile(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "delete profile", "unauthorized", err)
		return
	}

	err = h.profile.DeleteProfile(c.Request.Context(), userID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "delete profile", "service error", err)
		return
	}

	newSuccessResponse(c, http.StatusOK, "message", "profile deleted")
}

// getEmployeesList возвращает список всех активных сотрудников
func (h *Handler) getEmployeesList(c *gin.Context) {
	h.log.Debug("fetching employees list")

	users, err := h.profile.GetEmployeesList(c.Request.Context())
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getEmployeesList", "service error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"employees": users})
}
