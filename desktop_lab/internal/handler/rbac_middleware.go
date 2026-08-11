package handler

import (
	"desktop_lab/internal/models"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// permissionMiddleware создает middleware для проверки прав доступа на основе ролей
func (h *Handler) permissionMiddleware(requiredPermissions ...models.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем роль пользователя из контекста
		roleRaw, exists := c.Get(roleKey)
		if !exists {
			h.newErrorResponse(c, http.StatusUnauthorized, "permission denied", "role not found in context", nil)
			c.Abort()
			return
		}

		role, ok := roleRaw.(models.Role)
		if !ok {
			h.newErrorResponse(c, http.StatusInternalServerError, "permission denied", "invalid role type in context", nil)
			c.Abort()
			return
		}

		// Проверяем наличие всех требуемых разрешений
		for _, perm := range requiredPermissions {
			if !role.HasPermission(perm) {
				h.log.Warn("permission denied",
					zap.String("role", string(role)),
					zap.String("required_permission", string(perm)),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)
				h.newErrorResponse(c, http.StatusForbidden, "permission denied", "insufficient permissions", nil)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// requireRoleMiddleware создает middleware для проверки конкретной роли
func (h *Handler) requireRoleMiddleware(allowedRoles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем роль пользователя из контекста
		roleRaw, exists := c.Get(roleKey)
		if !exists {
			h.newErrorResponse(c, http.StatusUnauthorized, "access denied", "role not found in context", nil)
			c.Abort()
			return
		}

		role, ok := roleRaw.(models.Role)
		if !ok {
			h.newErrorResponse(c, http.StatusInternalServerError, "access denied", "invalid role type in context", nil)
			c.Abort()
			return
		}

		// Проверяем, входит ли роль в список разрешенных
		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				c.Next()
				return
			}
		}

		h.log.Warn("role access denied",
			zap.String("user_role", string(role)),
			zap.Strings("allowed_roles", rolesToStrings(allowedRoles)),
			zap.String("path", c.Request.URL.Path),
		)
		h.newErrorResponse(c, http.StatusForbidden, "access denied", "insufficient role privileges", nil)
		c.Abort()
	}
}

// rolesToStrings преобразует slice ролей в slice строк
func rolesToStrings(roles []models.Role) []string {
	result := make([]string, len(roles))
	for i, role := range roles {
		result[i] = string(role)
	}
	return result
}

// getRoleFromContext извлекает роль из контекста Gin
func getRoleFromContext(c *gin.Context) (models.Role, error) {
	roleRaw, exists := c.Get(roleKey)
	if !exists {
		return "", models.ErrNotFound
	}

	role, ok := roleRaw.(models.Role)
	if !ok {
		return "", fmt.Errorf("role not found")
	}

	return role, nil
}

// hasPermission проверяет, есть ли у пользователя определенное разрешение
func hasPermission(c *gin.Context, permission models.Permission) bool {
	role, err := getRoleFromContext(c)
	if err != nil {
		return false
	}
	return role.HasPermission(permission)
}

// hasAnyPermission проверяет, есть ли у пользователя хотя бы одно из разрешений
func hasAnyPermission(c *gin.Context, permissions ...models.Permission) bool {
	role, err := getRoleFromContext(c)
	if err != nil {
		return false
	}
	return role.HasAnyPermission(permissions...)
}

// isAdmin проверяет, является ли пользователь администратором
func isAdmin(c *gin.Context) bool {
	role, err := getRoleFromContext(c)
	if err != nil {
		return false
	}
	return role == models.RoleAdmin
}

// isEngineer проверяет, является ли пользователь инженером или админом
func isEngineer(c *gin.Context) bool {
	role, err := getRoleFromContext(c)
	if err != nil {
		return false
	}
	return role == models.RoleEngineer || role == models.RoleAdmin
}

// isTechnician проверяет, является ли пользователь техником, инженером или админом
func isTechnician(c *gin.Context) bool {
	role, err := getRoleFromContext(c)
	if err != nil {
		return false
	}
	return role == models.RoleTechnician || role == models.RoleEngineer || role == models.RoleAdmin
}

// isClient проверяет, является ли пользователь клиентом (или имеет более высокие права)
func isClient(c *gin.Context) bool {
	role, err := getRoleFromContext(c)
	if err != nil {
		return false
	}
	// Клиент может быть только клиентом - у него минимальные права
	return role == models.RoleClient
}
