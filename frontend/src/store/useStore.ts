import { create } from 'zustand'
import { devtools } from 'zustand/middleware'

export interface Message {
  id: string
  content: string
  role: 'user' | 'assistant'
  type: 'text' | 'code' | 'audio'
  language?: string
  timestamp: Date
}

export interface ConversationState {
  messages: Message[]
  isRecording: boolean
  isConnected: boolean
  currentSession: string | null
  audioLevel: number
  isProcessing: boolean
}

export interface ConversationActions {
  addMessage: (message: Omit<Message, 'id' | 'timestamp'>) => void
  setRecording: (recording: boolean) => void
  setConnected: (connected: boolean) => void
  setCurrentSession: (session: string | null) => void
  setAudioLevel: (level: number) => void
  setProcessing: (processing: boolean) => void
  clearMessages: () => void
  updateMessage: (id: string, updates: Partial<Message>) => void
}

export const useConversationStore = create<ConversationState & ConversationActions>()(
  devtools(
    (set) => ({
      // State
      messages: [],
      isRecording: false,
      isConnected: false,
      currentSession: null,
      audioLevel: 0,
      isProcessing: false,

      // Actions
      addMessage: (message) => {
        const newMessage: Message = {
          ...message,
          id: crypto.randomUUID(),
          timestamp: new Date(),
        }
        set((state) => ({
          messages: [...state.messages, newMessage],
        }))
      },

      setRecording: (recording) => {
        set({ isRecording: recording })
      },

      setConnected: (connected) => {
        set({ isConnected: connected })
      },

      setCurrentSession: (session) => {
        set({ currentSession: session })
      },

      setAudioLevel: (level) => {
        set({ audioLevel: level })
      },

      setProcessing: (processing) => {
        set({ isProcessing: processing })
      },

      clearMessages: () => {
        set({ messages: [] })
      },

      updateMessage: (id, updates) => {
        set((state) => ({
          messages: state.messages.map((msg) =>
            msg.id === id ? { ...msg, ...updates } : msg
          ),
        }))
      },
    }),
    {
      name: 'conversation-store',
    }
  )
)
