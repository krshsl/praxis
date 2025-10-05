import { create } from 'zustand'
import { devtools } from 'zustand/middleware'
import { authService, type User } from 'services/auth'

interface AuthState {
  user: User | null
  loading: boolean
  isAuthChecked: boolean
}

interface AuthActions {
  setUser: (user: User | null) => void
  setLoading: (loading: boolean) => void
  setIsAuthChecked: (isAuthChecked: boolean) => void
  login: (email: string, password: string) => Promise<{ data: any; error: null } | { data: null; error: any }>
  signUp: (name: string, email: string, password: string) => Promise<{ data: any; error: null } | { data: null; error: any }>
  signOut: () => Promise<{ error: null } | { error: any }>
  checkAuth: () => Promise<boolean>
  clearAuth: () => void
}

export const useAuth = create<AuthState & AuthActions>()(
  devtools(
    (set, get) => ({
      // State
      user: null,
      loading: true,
      isAuthChecked: false,

      // Actions
      setUser: (user) => set({ user }),
      setLoading: (loading) => set({ loading }),
      setIsAuthChecked: (isAuthChecked) => set({ isAuthChecked }),

      login: async (email: string, password: string) => {
        try {
          set({ loading: true })
          const response = await authService.login(email, password)
          set({ user: response.user, loading: false, isAuthChecked: true })
          return { data: response, error: null }
        } catch (error) {
          set({ loading: false, isAuthChecked: true })
          return { data: null, error }
        }
      },

      signUp: async (name: string, email: string, password: string) => {
        try {
          set({ loading: true })
          const response = await authService.signUp(email, password, name)
          set({ user: response.user, loading: false, isAuthChecked: true })
          return { data: response, error: null }
        } catch (error) {
          set({ loading: false, isAuthChecked: true })
          return { data: null, error }
        }
      },

      signOut: async () => {
        try {
          set({ loading: true })
          await authService.logout()
          set({ user: null, loading: false })
          return { error: null }
        } catch (error) {
          set({ loading: false })
          return { error: { message: error instanceof Error ? error.message : 'Logout failed' } }
        }
      },

      checkAuth: async () => {
        const state = get()
        if (state.isAuthChecked) {
          return state.user !== null
        }

        try {
          set({ loading: true })
          
          const timeoutPromise = new Promise<boolean>((_, reject) => 
            setTimeout(() => reject(new Error('Authentication timeout')), 10000)
          )
          
          const authPromise = authService.checkAuth()
          const isAuthenticated = await Promise.race([authPromise, timeoutPromise])
          
          if (isAuthenticated) {
            const currentUser = authService.getCurrentUser()
            set({ user: currentUser, loading: false, isAuthChecked: true })
          } else {
            set({ user: null, loading: false, isAuthChecked: true })
          }
          
          return isAuthenticated
        } catch (error) {
          console.error('Auth check failed:', error)
          set({ user: null, loading: false, isAuthChecked: true })
          return false
        }
      },

      clearAuth: () => {
        set({ user: null, loading: false, isAuthChecked: true })
      },
    }),
    {
      name: 'auth-store',
    }
  )
)

// Selectors for common use cases
export const useUser = () => useAuth((state) => state.user)
export const useAuthLoading = () => useAuth((state) => state.loading)
export const useIsAuthenticated = () => useAuth((state) => state.user !== null)
export const useIsAuthChecked = () => useAuth((state) => state.isAuthChecked)

// Individual action selectors to avoid object recreation
export const useLogin = () => useAuth((state) => state.login)
export const useSignUp = () => useAuth((state) => state.signUp)
export const useSignOut = () => useAuth((state) => state.signOut)
export const useCheckAuth = () => useAuth((state) => state.checkAuth)
export const useClearAuth = () => useAuth((state) => state.clearAuth)
