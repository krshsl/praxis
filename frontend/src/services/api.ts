import axios from 'axios'
import type { AxiosInstance, AxiosResponse, AxiosError } from 'axios'

// API Configuration
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'

// Types
export interface User {
  id: string
  email: string
  full_name: string
  avatar_url: string
  role: string
}

export interface Agent {
  id: string
  user_id?: string
  name: string
  description: string
  personality: string
  industry?: string
  level?: string
  is_public: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface Session {
  id: string
  user_id: string
  agent_id: string
  status: 'active' | 'completed' | 'abandoned'
  started_at: string
  ended_at?: string
  duration: number
  agent?: Agent
  created_at: string
  updated_at: string
}

export interface Transcript {
  id: string
  session_id: string
  turn_order: number
  speaker: 'user' | 'agent'
  content: string
  timestamp: string
  created_at: string
  updated_at: string
}

export interface Summary {
  id: string
  session_id: string
  summary: string
  strengths?: string
  weaknesses?: string
  recommendations?: string
  overall_score: number
  created_at: string
  updated_at: string
}

export interface Score {
  id: string
  session_id: string
  metric: string
  score: number
  max_score: number
  weight: number
  created_at: string
  updated_at: string
}

export interface AuthResponse {
  user: User
  message: string
}

export interface ApiError {
  message: string
  status?: number
}

// Create axios instance with default configuration
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  withCredentials: true, // Important for cookie-based auth
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor for logging
apiClient.interceptors.request.use(
  (config) => {
    return config
  },
  (error: any) => {
    console.error('[API] Request error:', error)
    return Promise.reject(error)
  }
)

// Response interceptor for error handling and token refresh
apiClient.interceptors.response.use(
  (response: AxiosResponse) => {
    return response
  },
  async (error: AxiosError) => {
    const originalRequest = error.config

    // Handle 401 errors (unauthorized) - but exclude auth endpoints and health check to prevent loops
    const isAuthEndpoint = originalRequest?.url?.includes('/auth/')
    const isHealthEndpoint = originalRequest?.url?.includes('/health')
    
    if (error.response?.status === 401 && originalRequest && !(originalRequest as any)._retry && !isAuthEndpoint && !isHealthEndpoint) {
      (originalRequest as any)._retry = true

      try {
        // Try to refresh the token
        await apiClient.post('/auth/refresh')
        
        // Retry the original request
        return apiClient(originalRequest)
      } catch (refreshError) {
        // Refresh failed, redirect to login or handle as needed
        console.error('[API] Token refresh failed:', refreshError)
        
        delete apiClient.defaults.headers.common['Authorization']
        
        window.location.href = '/login'
        return Promise.reject(refreshError)
      }
    }

    // Handle other errors
    const apiError: ApiError = {
      message: (error.response?.data as any)?.message || error.message || 'An error occurred',
      status: error.response?.status,
    }

    console.error('[API] Response error:', apiError)
    return Promise.reject(apiError)
  }
)

// API Service Class
class ApiService {
  // Authentication methods
  async getCurrentUser(): Promise<{ user: User }> {
    const response = await apiClient.get<{ user: User }>('/auth/me')
    return response.data
  }

  async syncUser(): Promise<{ user: User }> {
    const response = await apiClient.post<{ user: User }>('/auth/sync')
    return response.data
  }

  // Agent methods
  async getAgents(): Promise<{ agents: Agent[] }> {
    const response = await apiClient.get<{ agents: Agent[] }>('/agents')
    return response.data
  }

  async createAgent(agent: Partial<Agent>): Promise<{ agent: Agent }> {
    const response = await apiClient.post<{ agent: Agent }>('/agents', agent)
    return response.data
  }

  async getAgent(id: string): Promise<{ agent: Agent }> {
    const response = await apiClient.get<{ agent: Agent }>(`/agents/${id}`)
    return response.data
  }

  async updateAgent(id: string, agent: Partial<Agent>): Promise<{ agent: Agent }> {
    const response = await apiClient.put<{ agent: Agent }>(`/agents/${id}`, agent)
    return response.data
  }

  async deleteAgent(id: string): Promise<void> {
    await apiClient.delete(`/agents/${id}`)
  }

  // Session methods
  async getSessions(): Promise<{ sessions: Session[] }> {
    const response = await apiClient.get<{ sessions: Session[] }>('/sessions')
    return response.data
  }

  async createSession(agentId: string): Promise<{ session: Session }> {
    const response = await apiClient.post<{ session: Session }>('/sessions', { agent_id: agentId })
    return response.data
  }

  async getSession(id: string): Promise<{ session: Session }> {
    const response = await apiClient.get<{ session: Session }>(`/sessions/${id}`)
    return response.data
  }

  async endSession(id: string): Promise<{ session: Session }> {
    const response = await apiClient.put<{ session: Session }>(`/sessions/${id}/end`)
    return response.data
  }

  async deleteSession(id: string): Promise<void> {
    await apiClient.delete(`/sessions/${id}`)
  }

  async bulkDeleteSessions(sessionIds: string[]): Promise<{ message: string; deleted_count: number }> {
    const response = await apiClient.delete<{ message: string; deleted_count: number }>('/sessions/bulk', {
      data: { session_ids: sessionIds }
    })
    return response.data
  }

  // Transcript methods
  async getTranscripts(sessionId: string): Promise<{ transcripts: Transcript[] }> {
    const response = await apiClient.get<{ transcripts: Transcript[] }>(`/transcripts/session/${sessionId}`)
    return response.data
  }

  async addTranscript(transcript: Partial<Transcript>): Promise<{ transcript: Transcript }> {
    const response = await apiClient.post<{ transcript: Transcript }>('/transcripts', transcript)
    return response.data
  }

  // Summary methods
  async getSummary(sessionId: string): Promise<{ summary: Summary; status?: string }> {
    const response = await apiClient.get<{ summary: Summary; status?: string }>(`/summaries/session/${sessionId}`)
    return response.data
  }

  async createSummary(summary: Partial<Summary>): Promise<{ summary: Summary }> {
    const response = await apiClient.post<{ summary: Summary }>('/summaries', summary)
    return response.data
  }

  // Score methods
  async getScores(sessionId: string): Promise<{ scores: Score[] }> {
    const response = await apiClient.get<{ scores: Score[] }>(`/scores/session/${sessionId}`)
    return response.data
  }

  async addScore(score: Partial<Score>): Promise<{ score: Score }> {
    const response = await apiClient.post<{ score: Score }>('/scores', score)
    return response.data
  }

  // WebSocket URL for authenticated connections
  getWebSocketUrl(): string {
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsHost = import.meta.env.VITE_WS_URL || 'localhost:8080'
    return `${wsProtocol}//${wsHost}/api/v1/ws`
  }

  // Generic API methods
  async get<T>(url: string, config?: any): Promise<T> {
    const response = await apiClient.get<T>(url, config)
    return response.data
  }

  async post<T>(url: string, data?: any, config?: any): Promise<T> {
    const response = await apiClient.post<T>(url, data, config)
    return response.data
  }

  async put<T>(url: string, data?: any, config?: any): Promise<T> {
    const response = await apiClient.put<T>(url, data, config)
    return response.data
  }

  async delete<T>(url: string, config?: any): Promise<T> {
    const response = await apiClient.delete<T>(url, config)
    return response.data
  }

  // File upload method
  async uploadFile<T>(url: string, file: File, onProgress?: (progress: number) => void): Promise<T> {
    const formData = new FormData()
    formData.append('file', file)

    const config = {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: (progressEvent: any) => {
        if (onProgress && progressEvent.total) {
          const progress = Math.round((progressEvent.loaded * 100) / progressEvent.total)
          onProgress(progress)
        }
      },
    }

    const response = await apiClient.post<T>(url, formData, config)
    return response.data
  }

  // Health check
  async healthCheck(): Promise<{ status: string; database: string }> {
    const response = await apiClient.get<{ status: string; database: string }>('/health')
    return response.data
  }
}

// Export singleton instance
export const apiService = new ApiService()

// Export axios instance for advanced usage
export { apiClient }

// Export types
export type { AxiosResponse, AxiosError }
