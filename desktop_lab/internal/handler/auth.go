package handler

import (
	"desktop_lab/internal/models"
	"desktop_lab/pkg/valid"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) initAuthRoutes(api *gin.RouterGroup) {
	h.log.Debug("Init auth routes")
	auth := api.Group("/auth")
	{
		auth.POST("/sign-up", h.signUp)
		auth.POST("/sign-in", h.signIn)
		auth.GET("/logout", h.logout)
		auth.GET("/refresh", h.refresh)
		auth.GET("/me", h.authMiddleware, h.getMe)
	}
}

func (h *Handler) signUp(c *gin.Context) {
	var user models.CreateUserRequest

	if err := c.ShouldBindJSON(&user); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "sign up", "invalid request body", err)
		return
	}

	if err := valid.ValidateStruct(user); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "sign up", "invalid request body", err)
		return
	}

	// Запрещаем регистрацию с ролью администратора через публичный API
	if user.Role == models.RoleAdmin {
		h.newErrorResponse(c, http.StatusForbidden, "sign up", "admin registration is not allowed", nil)
		return
	}

	// Если роль не указана или невалидна, устанавливаем роль по умолчанию (client)
	if !user.Role.IsValid() {
		user.Role = models.RoleClient
	}

	userID, err := h.auth.SignUp(c.Request.Context(), user)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "sign up", "service error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"user_id": userID})
}

func (h *Handler) signIn(c *gin.Context) {
	var user models.SignInRequest

	if err := c.ShouldBindJSON(&user); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "sign in", "invalid request body", err)
		return
	}

	if err := valid.ValidateStruct(user); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "sign in", "invalid request body", err)
		return
	}

	token, err := h.auth.SignIn(c.Request.Context(), user)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "sign in", "service error", err)
		return
	}

	c.SetCookie(refreshToken, token.RefreshToken, int(h.refreshTokenTTL.Seconds()), "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{accessToken: token.AccessToken})
}

func (h *Handler) logout(c *gin.Context) {
	tokenID, err := getAccessToken(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "logout", "invalid token", err)
		return
	}

	if err := h.auth.Logout(c.Request.Context(), tokenID); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "logout", "service error", err)
		return
	}

	c.SetCookie(refreshToken, "", 0, "/", "", false, true)
	c.JSON(http.StatusOK, "logout")
}

func (h *Handler) refresh(c *gin.Context) {
	refreshTkn, err := getRefreshToken(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "refresh", "invalid refresh token", err)
		return
	}

	token, err := h.auth.RefreshToken(c.Request.Context(), refreshTkn)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "refresh", "service error", err)
		return
	}

	c.SetCookie(refreshToken, token.RefreshToken, int(h.refreshTokenTTL.Seconds()), "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{accessToken: token.AccessToken})
}

// getMe возвращает информацию о текущем пользователе
func (h *Handler) getMe(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusUnauthorized, "get me", "unauthorized", err)
		return
	}

	user, err := h.auth.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "get me", "service error", err)
		return
	}

	// Возвращаем только безопасные данные (без пароля)
	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"role":       user.Role,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	})
}
