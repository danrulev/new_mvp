package handler

import (
	"desktop_lab/internal/models"
	"desktop_lab/pkg/valid"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) initOrganizationRoutes(api *gin.RouterGroup) {
	h.log.Debug("Init organization routes")
	// Организации требуют аутентификации
	organizations := api.Group("/organizations")
	organizations.Use(h.authMiddleware)
	{
		// Создание организации - только админ
		organizations.POST("/", h.permissionMiddleware(models.PermOrganizationCreate), h.createOrganization)
		// Чтение организаций - все аутентифицированные
		organizations.GET("/:id", h.permissionMiddleware(models.PermOrganizationRead), h.getOrganizationByID)
		organizations.GET("/name/:name", h.permissionMiddleware(models.PermOrganizationRead), h.getOrganizationByName)
		organizations.GET("/", h.permissionMiddleware(models.PermOrganizationRead), h.listOrganizations)
		// Обновление и удаление - только админ
		organizations.PUT("/:id", h.permissionMiddleware(models.PermOrganizationUpdate), h.updateOrganization)
		organizations.DELETE("/:id", h.permissionMiddleware(models.PermOrganizationDelete), h.deleteOrganization)

		// Organization Users routes - управление пользователями организации
		// Создание пользователя в организации - только админ
		organizations.POST("/:id/users", h.permissionMiddleware(models.PermUserCreate), h.createOrganizationUser)
		// Чтение пользователей - все аутентифицированные
		organizations.GET("/:id/users/:userID", h.permissionMiddleware(models.PermUserRead), h.getOrganizationUserByID)
		organizations.GET("/:id/users", h.permissionMiddleware(models.PermUserRead), h.listOrganizationUsers)
		organizations.GET("/:id/users/role/:role", h.permissionMiddleware(models.PermUserRead), h.getOrganizationUserByRole)
		// Обновление и удаление пользователей - только админ
		organizations.PUT("/:id/users/:userID", h.permissionMiddleware(models.PermUserUpdate), h.updateOrganizationUser)
		organizations.DELETE("/:id/users/:userID", h.permissionMiddleware(models.PermUserDelete), h.deleteOrganizationUser)

		// Organization Tests routes - тесты организации (только админ)
		organizations.POST("/:id/tests", h.requireRoleMiddleware(models.RoleAdmin), h.createOrganizationTest)
		organizations.GET("/tests/:testID", h.requireRoleMiddleware(models.RoleAdmin), h.getOrganizationTest)
		organizations.GET("/tests", h.requireRoleMiddleware(models.RoleAdmin), h.listOrganizationTests)
		organizations.PUT("/:id/tests/:testID", h.requireRoleMiddleware(models.RoleAdmin), h.updateOrganizationTest)
		organizations.DELETE("/:id/tests/:testID", h.requireRoleMiddleware(models.RoleAdmin), h.deleteOrganizationTest)
	}
}

func (h *Handler) createOrganization(c *gin.Context) {
	var req models.CreateOrganizationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create organization", "invalid request body", err)
		return
	}

	if err := valid.ValidateStruct(req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create organization", "invalid request body", err)
		return
	}

	id, err := h.organization.CreateOrganization(c.Request.Context(), req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "create organization", "service error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (h *Handler) getOrganizationByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get organization by id", "id param is empty", nil)
		return
	}

	org, err := h.organization.GetOrganizationByID(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get organization by id", "service error", err)
		return
	}

	c.JSON(http.StatusOK, org)
}

func (h *Handler) getOrganizationByName(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get organization by name", "name param is empty", nil)
		return
	}

	org, err := h.organization.GetOrganizationByName(c.Request.Context(), name)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get organization by name", "service error", err)
		return
	}

	c.JSON(http.StatusOK, org)
}

func (h *Handler) listOrganizations(c *gin.Context) {
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

	orgs, err := h.organization.ListOrganizations(c.Request.Context(), limit, offset)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "list organizations", "service error", err)
		return
	}

	c.JSON(http.StatusOK, orgs)
}

func (h *Handler) updateOrganization(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "update organization", "id param is empty", nil)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "update organization", "user not authenticated", err)
		return
	}

	var req models.UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "update organization", "invalid request body", err)
		return
	}

	org, err := h.organization.UpdateOrganization(c.Request.Context(), userID, id, req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "update organization", "service error", err)
		return
	}

	c.JSON(http.StatusOK, org)
}

func (h *Handler) deleteOrganization(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "delete organization", "id param is empty", nil)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "delete organization", "user not authenticated", err)
		return
	}

	if err := h.organization.DeleteOrganization(c.Request.Context(), userID, id); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "delete organization", "service error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "organization deleted"})
}

func (h *Handler) createOrganizationUser(c *gin.Context) {
	organizationID := c.Param("id")
	if organizationID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "create organization user", "organization id param is empty", nil)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "create organization user", "user not authenticated", err)
		return
	}

	var req models.CreateOrganizationUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create organization user", "invalid request body", err)
		return
	}

	if err := valid.ValidateStruct(req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create organization user", "invalid request body", err)
		return
	}

	id, err := h.organization.CreateOrganizationUser(c.Request.Context(), userID, organizationID, req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "create organization user", "service error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (h *Handler) getOrganizationUserByID(c *gin.Context) {
	organizationID := c.Param("id")
	userID := c.Param("userID")

	if organizationID == "" || userID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get organization user by id", "id params are empty", nil)
		return
	}

	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "get organization user by id", "user not authenticated", err)
		return
	}

	orgUser, err := h.organization.GetOrganizationUserByID(c.Request.Context(), authUserID, organizationID, userID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get organization user by id", "service error", err)
		return
	}

	c.JSON(http.StatusOK, orgUser)
}

func (h *Handler) listOrganizationUsers(c *gin.Context) {
	organizationID := c.Param("id")
	if organizationID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "list organization users", "organization id param is empty", nil)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "list organization users", "user not authenticated", err)
		return
	}

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

	users, err := h.organization.ListOrganizationUsers(c.Request.Context(), userID, organizationID, limit, offset)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "list organization users", "service error", err)
		return
	}

	c.JSON(http.StatusOK, users)
}

func (h *Handler) getOrganizationUserByRole(c *gin.Context) {
	organizationID := c.Param("id")
	role := c.Param("role")

	if organizationID == "" || role == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get organization user by role", "id or role params are empty", nil)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "get organization user by role", "user not authenticated", err)
		return
	}

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

	users, err := h.organization.GetOrganizationUserByRole(c.Request.Context(), userID, organizationID, role, limit, offset)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get organization user by role", "service error", err)
		return
	}

	c.JSON(http.StatusOK, users)
}

func (h *Handler) updateOrganizationUser(c *gin.Context) {
	organizationID := c.Param("id")
	userID := c.Param("userID")

	if organizationID == "" || userID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "update organization user", "id params are empty", nil)
		return
	}

	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "update organization user", "user not authenticated", err)
		return
	}

	var req struct {
		Role *string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "update organization user", "invalid request body", err)
		return
	}

	orgUser, err := h.organization.UpdateOrganizationUser(c.Request.Context(), authUserID, organizationID, userID, req.Role)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "update organization user", "service error", err)
		return
	}

	c.JSON(http.StatusOK, orgUser)
}

func (h *Handler) deleteOrganizationUser(c *gin.Context) {
	organizationID := c.Param("id")
	userID := c.Param("userID")

	if organizationID == "" || userID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "delete organization user", "id params are empty", nil)
		return
	}

	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "delete organization user", "user not authenticated", err)
		return
	}

	if err := h.organization.DeleteOrganizationUser(c.Request.Context(), authUserID, organizationID, userID); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "delete organization user", "service error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "organization user deleted"})
}

func (h *Handler) createOrganizationTest(c *gin.Context) {
	organizationID := c.Param("id")
	if organizationID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "create organization test", "organization id param is empty", nil)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "create organization test", "user not authenticated", err)
		return
	}

	var req models.CreateOrganizationTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create organization test", "invalid request body", err)
		return
	}

	if err := valid.ValidateStruct(req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "create organization test", "invalid request body", err)
		return
	}

	id, err := h.organization.CreateOrganizationTest(c.Request.Context(), userID, organizationID, req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "create organization test", "service error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (h *Handler) getOrganizationTest(c *gin.Context) {
	testID := c.Param("testID")
	if testID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "get organization test", "test id param is empty", nil)
		return
	}

	test, err := h.organization.GetOrganizationTest(c.Request.Context(), testID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get organization test", "service error", err)
		return
	}

	c.JSON(http.StatusOK, test)
}

func (h *Handler) listOrganizationTests(c *gin.Context) {
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

	tests, err := h.organization.ListOrganizationTests(c.Request.Context(), limit, offset)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "list organization tests", "service error", err)
		return
	}

	c.JSON(http.StatusOK, tests)
}

func (h *Handler) updateOrganizationTest(c *gin.Context) {
	organizationID := c.Param("id")
	testID := c.Param("testID")

	if organizationID == "" || testID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "update organization test", "id params are empty", nil)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "update organization test", "user not authenticated", err)
		return
	}

	var req models.UpdateOrganizationTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "update organization test", "invalid request body", err)
		return
	}

	test, err := h.organization.UpdateOrganizationTest(c.Request.Context(), userID, organizationID, testID, req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "update organization test", "service error", err)
		return
	}

	c.JSON(http.StatusOK, test)
}

func (h *Handler) deleteOrganizationTest(c *gin.Context) {
	organizationID := c.Param("id")
	testID := c.Param("testID")

	if organizationID == "" || testID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "delete organization test", "id params are empty", nil)
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "delete organization test", "user not authenticated", err)
		return
	}

	if err := h.organization.DeleteOrganizationTest(c.Request.Context(), userID, organizationID, testID); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "delete organization test", "service error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "organization test deleted"})
}
