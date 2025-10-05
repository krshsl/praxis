package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/krshsl/praxis/backend/models"
	"github.com/krshsl/praxis/backend/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo            *repository.GORMRepository
	jwtSecret       []byte
	accessExpiry    time.Duration
	refreshExpiry   time.Duration
	permanentExpiry time.Duration
}

type CookieClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type AuthResponse struct {
	User           *models.User `json:"user"`
	AccessToken    string       `json:"access_token,omitempty"`
	RefreshToken   string       `json:"refresh_token,omitempty"`
	PermanentToken string       `json:"permanent_token,omitempty"`
}

func NewAuthService(repo *repository.GORMRepository, jwtSecret string) *AuthService {
	return &AuthService{
		repo:            repo,
		jwtSecret:       []byte(jwtSecret),
		accessExpiry:    5 * time.Minute,     // 5 minutes
		refreshExpiry:   7 * 24 * time.Hour,  // 7 days
		permanentExpiry: 30 * 24 * time.Hour, // 30 days
	}
}

// generateSecureToken generates a cryptographically secure random token
func (s *AuthService) generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashToken creates a SHA256 hash of the token for secure storage
func (s *AuthService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Login authenticates user and creates tokens
func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	// Get user by email
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	permanentToken, err := s.generatePermanentToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate permanent token: %w", err)
	}

	// Store tokens in database
	if err := s.storeTokens(ctx, user.ID, refreshToken, permanentToken); err != nil {
		return nil, fmt.Errorf("failed to store tokens: %w", err)
	}

	slog.Info("User logged in successfully", "user_id", user.ID, "email", user.Email)
	return &AuthResponse{
		User:           user,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		PermanentToken: permanentToken,
	}, nil
}

// Signup creates a new user
func (s *AuthService) Signup(ctx context.Context, email, password, fullName string) (*AuthResponse, error) {
	// Check if user already exists
	existingUser, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, fmt.Errorf("user already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Email:    email,
		Password: string(hashedPassword),
		FullName: fullName,
		Role:     "user",
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	permanentToken, err := s.generatePermanentToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate permanent token: %w", err)
	}

	// Store tokens in database
	if err := s.storeTokens(ctx, user.ID, refreshToken, permanentToken); err != nil {
		return nil, fmt.Errorf("failed to store tokens: %w", err)
	}

	slog.Info("User signed up successfully", "user_id", user.ID, "email", user.Email)
	return &AuthResponse{
		User:           user,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		PermanentToken: permanentToken,
	}, nil
}

// RefreshToken generates a new access token using refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	// Get refresh token from database
	tokenRecord, err := s.repo.GetRefreshToken(ctx, s.hashToken(refreshToken))
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	if tokenRecord == nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Get user
	user, err := s.repo.GetUserByID(ctx, tokenRecord.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Generate new access token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	slog.Info("Access token refreshed", "user_id", user.ID)
	return &AuthResponse{
		User:        user,
		AccessToken: accessToken,
	}, nil
}

// VerifyPermanentToken verifies permanent token and generates new access token
func (s *AuthService) VerifyPermanentToken(ctx context.Context, permanentToken string) (*AuthResponse, error) {
	// Get permanent token from database
	tokenRecord, err := s.repo.GetPermanentToken(ctx, s.hashToken(permanentToken))
	if err != nil {
		return nil, fmt.Errorf("failed to get permanent token: %w", err)
	}
	if tokenRecord == nil {
		return nil, fmt.Errorf("invalid permanent token")
	}

	// Get user
	user, err := s.repo.GetUserByID(ctx, tokenRecord.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Generate new access token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	slog.Info("Access token generated from permanent token", "user_id", user.ID)
	return &AuthResponse{
		User:        user,
		AccessToken: accessToken,
	}, nil
}

// Logout invalidates all tokens for the user
func (s *AuthService) Logout(ctx context.Context, userID string) error {
	if err := s.repo.DeleteAllUserTokens(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user tokens: %w", err)
	}

	slog.Info("User logged out", "user_id", userID)
	return nil
}

// VerifyAccessToken verifies and extracts user from access token
func (s *AuthService) VerifyAccessToken(ctx context.Context, token string) (*models.User, error) {
	claims := &CookieClaims{}

	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !parsedToken.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Get user from database to ensure they still exist
	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

// generateAccessToken creates a short-lived access token
func (s *AuthService) generateAccessToken(user *models.User) (string, error) {
	claims := &CookieClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// generateRefreshToken creates a long-lived refresh token
func (s *AuthService) generateRefreshToken(user *models.User) (string, error) {
	return s.generateSecureToken()
}

// generatePermanentToken creates a permanent token for security checks
func (s *AuthService) generatePermanentToken(user *models.User) (string, error) {
	return s.generateSecureToken()
}

// storeTokens stores refresh and permanent tokens in database
func (s *AuthService) storeTokens(ctx context.Context, userID, refreshToken, permanentToken string) error {
	// Store refresh token
	refreshTokenRecord := &models.RefreshToken{
		UserID:    userID,
		Token:     s.hashToken(refreshToken),
		ExpiresAt: time.Now().Add(s.refreshExpiry),
	}
	if err := s.repo.CreateRefreshToken(ctx, refreshTokenRecord); err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Store permanent token
	permanentTokenRecord := &models.PermanentToken{
		UserID: userID,
		Token:  s.hashToken(permanentToken),
	}
	if err := s.repo.CreatePermanentToken(ctx, permanentTokenRecord); err != nil {
		return fmt.Errorf("failed to store permanent token: %w", err)
	}

	return nil
}

// SetAuthCookies sets HTTP-only, secure cookies
func (s *AuthService) SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken, permanentToken string) {
	// Determine if we're in production (HTTPS) or development (HTTP)
	isProduction := os.Getenv("ENVIRONMENT") == "production"

	// Access token cookie (5 minutes)
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProduction,         // Only secure in production
		SameSite: http.SameSiteLaxMode, // More permissive for development
		MaxAge:   int(s.accessExpiry.Seconds()),
	})

	// Refresh token cookie (7 days)
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProduction,         // Only secure in production
		SameSite: http.SameSiteLaxMode, // More permissive for development
		MaxAge:   int(s.refreshExpiry.Seconds()),
	})

	// Permanent token cookie (30 days)
	http.SetCookie(w, &http.Cookie{
		Name:     "permanent_token",
		Value:    permanentToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProduction,         // Only secure in production
		SameSite: http.SameSiteLaxMode, // More permissive for development
		MaxAge:   int(s.permanentExpiry.Seconds()),
	})
}

// ClearAuthCookies clears all authentication cookies
func (s *AuthService) ClearAuthCookies(w http.ResponseWriter) {
	isProduction := os.Getenv("ENVIRONMENT") == "production"
	cookies := []string{"access_token", "refresh_token", "permanent_token"}

	for _, cookieName := range cookies {
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   isProduction,         // Only secure in production
			SameSite: http.SameSiteLaxMode, // More permissive for development
			MaxAge:   -1,
		})
	}
}

// GetTokenFromCookie extracts token from request cookies
func (s *AuthService) GetTokenFromCookie(r *http.Request, cookieName string) string {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// Middleware for cookie-based authentication
func (s *AuthService) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get access token from cookie
		accessToken := s.GetTokenFromCookie(r, "access_token")

		if accessToken != "" {
			// Verify access token
			user, err := s.VerifyAccessToken(r.Context(), accessToken)
			if err == nil {
				// Valid access token, proceed
				ctx := context.WithValue(r.Context(), "user", user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try to refresh using refresh token
		refreshToken := s.GetTokenFromCookie(r, "refresh_token")
		if refreshToken != "" {
			authResponse, err := s.RefreshToken(r.Context(), refreshToken)
			if err == nil {
				// Set new access token cookie
				s.SetAuthCookies(w, authResponse.AccessToken, "", "")

				// Add user to context and proceed
				ctx := context.WithValue(r.Context(), "user", authResponse.User)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try to use permanent token as last resort
		permanentToken := s.GetTokenFromCookie(r, "permanent_token")
		if permanentToken != "" {
			authResponse, err := s.VerifyPermanentToken(r.Context(), permanentToken)
			if err == nil {
				// Set new access token cookie
				s.SetAuthCookies(w, authResponse.AccessToken, "", "")

				// Add user to context and proceed
				ctx := context.WithValue(r.Context(), "user", authResponse.User)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// All authentication methods failed
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
