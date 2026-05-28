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

	userID, err := h.auth.SignUp(c.Request.Context(), user)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "sign up", "service error", err)
		return
	}

	newSuccessResponse(c, http.StatusOK, "id", userID)
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
	newSuccessResponse(c, http.StatusOK, accessToken, token.AccessToken)
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
	newSuccessResponse(c, http.StatusOK, "message", "logout")
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
	newSuccessResponse(c, http.StatusOK, accessToken, token.AccessToken)
}
