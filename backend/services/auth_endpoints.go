package services

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/krshsl/praxis/backend/models"
)

type AuthEndpoints struct {
	authService *AuthService
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

func NewAuthEndpoints(authService *AuthService) *AuthEndpoints {
	return &AuthEndpoints{
		authService: authService,
	}
}

func (e *AuthEndpoints) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", e.LoginHandler)
		r.Post("/signup", e.SignupHandler)
		r.Post("/refresh", e.RefreshHandler)
		r.Post("/logout", e.LogoutHandler)
		r.Get("/me", e.MeHandler)
	})
}

func (e *AuthEndpoints) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	authResponse, err := e.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		slog.Error("Login failed", "error", err, "email", req.Email)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Set cookies
	e.authService.SetAuthCookies(w, authResponse.AccessToken, authResponse.RefreshToken, authResponse.PermanentToken)

	// Return user info (without sensitive data)
	response := map[string]interface{}{
		"user": map[string]interface{}{
			"id":        authResponse.User.ID,
			"email":     authResponse.User.Email,
			"full_name": authResponse.User.FullName,
			"role":      authResponse.User.Role,
		},
		"message": "Login successful",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	slog.Info("User logged in", "user_id", authResponse.User.ID, "email", authResponse.User.Email)
}

func (e *AuthEndpoints) SignupHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Signup request received", "request", r.Body)

	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	authResponse, err := e.authService.Signup(r.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		slog.Error("Signup failed", "error", err, "email", req.Email)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set cookies
	e.authService.SetAuthCookies(w, authResponse.AccessToken, authResponse.RefreshToken, authResponse.PermanentToken)

	// Return user info (without sensitive data)
	response := map[string]interface{}{
		"user": map[string]interface{}{
			"id":        authResponse.User.ID,
			"email":     authResponse.User.Email,
			"full_name": authResponse.User.FullName,
			"role":      authResponse.User.Role,
		},
		"message": "Signup successful",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	slog.Info("User signed up", "user_id", authResponse.User.ID, "email", authResponse.User.Email)
}

func (e *AuthEndpoints) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken := e.authService.GetTokenFromCookie(r, "refresh_token")
	if refreshToken == "" {
		http.Error(w, "No refresh token provided", http.StatusUnauthorized)
		return
	}

	authResponse, err := e.authService.RefreshToken(r.Context(), refreshToken)
	if err != nil {
		slog.Error("Token refresh failed", "error", err)
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	// Set new access token cookie
	e.authService.SetAuthCookies(w, authResponse.AccessToken, "", "")

	response := map[string]interface{}{
		"message": "Token refreshed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	slog.Info("Token refreshed", "user_id", authResponse.User.ID)
}

func (e *AuthEndpoints) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by middleware)
	user := r.Context().Value("user")
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Type assert to get user ID
	var userID string
	if authUser, ok := user.(*models.User); ok {
		userID = authUser.ID
	} else {
		http.Error(w, "Invalid user context", http.StatusInternalServerError)
		return
	}

	// Logout user (invalidate all tokens)
	if err := e.authService.Logout(r.Context(), userID); err != nil {
		slog.Error("Logout failed", "error", err, "user_id", userID)
		http.Error(w, "Logout failed", http.StatusInternalServerError)
		return
	}

	// Clear all cookies
	e.authService.ClearAuthCookies(w)

	response := map[string]interface{}{
		"message": "Logout successful",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	slog.Info("User logged out", "user_id", userID)
}

func (e *AuthEndpoints) MeHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by middleware)
	user := r.Context().Value("user")
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Type assert to get user
	authUser, ok := user.(*models.User)
	if !ok {
		http.Error(w, "Invalid user context", http.StatusInternalServerError)
		return
	}

	// Return user info (without sensitive data)
	response := map[string]interface{}{
		"user": map[string]interface{}{
			"id":        authUser.ID,
			"email":     authUser.Email,
			"full_name": authUser.FullName,
			"role":      authUser.Role,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
