// Backend-based authentication service
import { apiService } from 'services/api'
import type { User } from 'services/api'

// Re-export types for convenience
export type { User }

class AuthService {
  private user: User | null = null
  private isInitialized = false
  private isInitializing = false

  constructor() {
    // Don't auto-initialize - let Zustand store handle initialization
    // this.initializeAuth()
  }

  private async initializeAuth() {
    // Prevent multiple simultaneous initialization attempts
    if (this.isInitializing) {
      return
    }
    
    this.isInitializing = true
    
    try {
      // Check if user is authenticated by calling backend
      const response = await apiService.getCurrentUser()
      this.user = response.user
      this.isInitialized = true
    } catch {
      this.user = null
      this.isInitialized = true
      
      // Clear any stored auth data when unauthenticated
      this.clearStoredAuthData()
    } finally {
      this.isInitializing = false
    }
  }

  private clearStoredAuthData() {
    // Note: Auth tokens are stored as HttpOnly cookies by the backend
    // JavaScript cannot access or clear them - only the backend can
    // This method only clears any frontend state, not actual auth tokens
  }

  // Login method using backend API
  async login(email: string, password: string): Promise<{ user: User }> {
    const response = await apiService.post<{ user: User }>('/auth/login', {
      email,
      password,
    })
    
    this.user = response.user
    this.isInitialized = true
    return { user: this.user }
  }

  // Signup method using backend API
  async signUp(email: string, password: string, fullName: string): Promise<{ user: User }> {
    const response = await apiService.post<{ user: User }>('/auth/signup', {
      email,
      password,
      full_name: fullName,
    })
    
    this.user = response.user
    this.isInitialized = true
    return { user: this.user }
  }

  // Method to manually set user state (useful after successful auth)
  setUser(user: User | null): void {
    this.user = user
    this.isInitialized = true
  }

  async logout(): Promise<void> {
    try {
      // Call backend logout endpoint
      await apiService.post('/auth/logout')
    } catch {
      // Silent failure - still clear local state
    } finally {
      // Clear local state regardless of cleanup success
      this.user = null
      this.isInitialized = true
      
      // Clear all stored auth data
      this.clearStoredAuthData()
    }
  }

  getCurrentUser(): User | null {
    return this.user
  }

  isAuthenticated(): boolean {
    return this.user !== null
  }

  async checkAuth(): Promise<boolean> {
    // Wait for initialization to complete if it's not already done
    if (!this.isInitialized) {
      if (!this.isInitializing) {
        await this.initializeAuth()
      } else {
        // Wait for the ongoing initialization to finish
        while (this.isInitializing) {
          await new Promise(resolve => setTimeout(resolve, 50))
        }
      }
    }
    
    return this.isAuthenticated()
  }

  // Get WebSocket URL for authenticated connections
  getWebSocketUrl(): string {
    return apiService.getWebSocketUrl()
  }
}

export const authService = new AuthService()