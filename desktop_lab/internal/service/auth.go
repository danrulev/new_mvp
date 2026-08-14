package service

import (
	"context"
	"desktop_lab/internal/config"
	"desktop_lab/internal/models"
	"desktop_lab/pkg/hasher"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"go.uber.org/zap"
)

type AuthService struct {
	user   UserRepo
	token  TokenRepo
	hasher *hasher.Hasher
	cfg    config.AuthCfg

	log *zap.Logger
}

func NewAuthService(userRepo UserRepo, tokenRepo TokenRepo, cfg config.AuthCfg, log *zap.Logger) *AuthService {
	hasher := hasher.NewHasher()
	return &AuthService{
		user:   userRepo,
		token:  tokenRepo,
		hasher: hasher,
		cfg:    cfg,
		log:    log,
	}
}

func (s *AuthService) SignUp(ctx context.Context, req models.CreateUserRequest) (string, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "SignUp"),
		zap.String("name", req.Name),
		zap.String("email", req.Email),
	)
	log.Debug("creating new user")

	hash, err := s.hasher.GenerateHash(req.Password)
	if err != nil {
		log.Error("failed to generate hash", zap.Error(err))
		return "", err
	}

	user := models.User{
		ID:       uuid.New().String(),
		Email:    req.Email,
		Name:     req.Name,
		Password: hash,
		Role:     req.Role,
	}

	if err := s.user.Create(ctx, user); err != nil {
		log.Error("error creating user", zap.Error(err))
		return "", err
	}

	log.Debug("user created")

	return user.ID, nil
}

func (s *AuthService) SignIn(ctx context.Context, req models.SignInRequest) (models.TokenResponse, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "SignIn"),
		zap.String("email", req.Email))

	log.Debug("signing in")

	userID, hashedPassword, err := s.user.Credential(ctx, req.Email)
	if err != nil {
		log.Error("error fetching user credentials", zap.Error(err))
		return models.TokenResponse{}, err
	}

	if err := s.hasher.ComparePassword(hashedPassword, req.Password); err != nil {
		log.Error("invalid password", zap.Error(err))
		return models.TokenResponse{}, err
	}

	token, err := s.generateAndSaveTokens(ctx, userID)
	if err != nil {
		log.Error("failed to generate or save tokens",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return models.TokenResponse{}, err
	}
	log.Info("user signed in", zap.String("user_id", userID))

	return token, nil
}

func (a *AuthService) Logout(ctx context.Context, tokenID string) error {
	log := loggerWith(ctx, a.log, zap.String("service_name", "SignIn"))

	log.Debug("logout")
	if err := a.token.Delete(ctx, tokenID); err != nil {
		log.Error("failed to delete refresh token during logout",
			zap.Error(err),
		)
		return err
	}

	log.Info("user logged out successfully")

	return nil
}

func (a *AuthService) generateAndSaveTokens(ctx context.Context, userID string) (models.TokenResponse, error) {
	accessToken, refreshToken, err := a.generateTokens(userID)
	if err != nil {
		return models.TokenResponse{}, err
	}

	if err := a.token.Create(ctx, refreshToken); err != nil {
		return models.TokenResponse{}, err
	}

	return models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.ID,
	}, nil
}

func (a *AuthService) generateTokens(userID string) (string, models.Token, error) {
	accessToken, err := a.generateAccessToken(userID)
	if err != nil {
		return "", models.Token{}, err
	}

	refreshToken := a.generateRefreshToken(userID)
	return accessToken, refreshToken, nil
}

func (a *AuthService) generateAccessToken(userID string) (string, error) {
	tkn := jwt.New()
	if err := tkn.Set(jwt.SubjectKey, userID); err != nil {
		return "", fmt.Errorf("failed to set subject in token: %w", err)
	}

	if err := tkn.Set(jwt.ExpirationKey, time.Now().Add(a.cfg.AccessTokenTTL)); err != nil {
		return "", fmt.Errorf("failed to set expiration in token: %w", err)
	}

	if err := tkn.Set(jwt.IssuedAtKey, time.Now()); err != nil {
		return "", fmt.Errorf("failed to set issued at in token: %w", err)
	}

	accessToken, err := jwt.Sign(tkn, jwt.WithKey(jwa.HS256, []byte(a.cfg.JwtSecret)))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %s", err)
	}

	return string(accessToken), nil
}

func (a *AuthService) generateRefreshToken(userID string) models.Token {
	return models.Token{
		ID:        uuid.New().String(),
		UserID:    userID,
		ExpiresAt: models.TimeString(time.Now().Add(a.cfg.RefreshTokenTTL)),
	}
}

func (a *AuthService) ParseToken(ctx context.Context, accessToken string) (string, error) {
	log := loggerWith(ctx, a.log, zap.String("service_name", "ParseToken"))

	log.Debug("parsing token")
	verified, err := jwt.Parse([]byte(accessToken), jwt.WithKey(jwa.HS256, []byte(a.cfg.JwtSecret)))
	if err != nil {
		log.Debug("failed to parse or verify access token",
			zap.Error(err),
		)
		return "", fmt.Errorf("invalid token")
	}

	subject, ok := verified.Get(jwt.SubjectKey)
	if !ok {
		log.Debug("token missing 'sub' claim")
		return "", fmt.Errorf("invalid token")
	}

	userID, ok := subject.(string)
	if !ok {
		log.Debug("token 'sub' claim is not a string")
		return "", fmt.Errorf("invalid token")
	}

	log.Debug("token verified", zap.String("user_id", userID))

	return userID, nil
}

func (a *AuthService) RefreshToken(ctx context.Context, tokenID string) (models.TokenResponse, error) {
	log := loggerWith(ctx, a.log, zap.String("service_name", "RefreshToken"))
	tokenDB, err := a.token.TokenByID(ctx, tokenID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			log.Warn("refresh token not found (possible reuse or logout)",
				zap.String("token_id", tokenID),
			)
		} else {
			log.Error("failed to get refresh token from repo",
				zap.String("token_id", tokenID),
				zap.Error(err),
			)
		}
		return models.TokenResponse{}, err
	}

	if err := a.token.Delete(ctx, tokenID); err != nil {
		log.Error("failed to delete old refresh token",
			zap.String("token_id", tokenID),
			zap.Error(err),
		)
		return models.TokenResponse{}, err
	}

	if tokenDB.ExpiresAt.ToTime().Before(time.Now()) {
		log.Warn("attempt to refresh expired token",
			zap.String("token_id", tokenID),
			zap.Time("expires_at", tokenDB.ExpiresAt.ToTime()),
		)
		return models.TokenResponse{}, fmt.Errorf("token expired")
	}

	token, err := a.generateAndSaveTokens(ctx, tokenDB.UserID)
	if err != nil {
		log.Error("failed to generate new tokens during refresh",
			zap.String("user_id", tokenDB.UserID),
			zap.Error(err),
		)
		return models.TokenResponse{}, err
	}

	log.Info("token refreshed successfully",
		zap.String("user_id", tokenDB.UserID),
	)

	return token, nil
}

// GetUserByID получает пользователя по ID
func (s *AuthService) GetUserByID(ctx context.Context, userID string) (models.User, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetUserByID"),
		zap.String("user_id", userID),
	)

	log.Debug("fetching user by ID")

	user, err := s.user.GetByID(ctx, userID)
	if err != nil {
		log.Error("failed to fetch user", zap.Error(err))
		return models.User{}, err
	}

	log.Debug("user fetched successfully")
	return user, nil
}
