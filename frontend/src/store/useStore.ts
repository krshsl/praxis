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
  isUserSpeaking: boolean
  isAISpeaking: boolean
  isThinking: boolean
  thinkingTimeRemaining: number
  speakingTimeRemaining: number
  sessionEnded: boolean
  sessionEndReason: string | null
}

export interface ConversationActions {
  addMessage: (message: Omit<Message, 'id' | 'timestamp'>) => void
  setRecording: (recording: boolean) => void
  setConnected: (connected: boolean) => void
  setCurrentSession: (session: string | null) => void
  setAudioLevel: (level: number) => void
  setProcessing: (processing: boolean) => void
  setUserSpeaking: (speaking: boolean) => void
  setAISpeaking: (speaking: boolean) => void
  setThinking: (thinking: boolean) => void
  setThinkingTimeRemaining: (time: number) => void
  setSpeakingTimeRemaining: (time: number) => void
  clearMessages: () => void
  updateMessage: (id: string, updates: Partial<Message>) => void
  setSessionEnded: (ended: boolean, reason?: string) => void
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
      isUserSpeaking: false,
      isAISpeaking: false,
      isThinking: false,
      thinkingTimeRemaining: 0,
      speakingTimeRemaining: 0,
      sessionEnded: false,
      sessionEndReason: null,

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

      setUserSpeaking: (speaking) => {
        set({ isUserSpeaking: speaking })
      },

      setAISpeaking: (speaking) => {
        set({ isAISpeaking: speaking })
      },

      setThinking: (thinking) => {
        set({ isThinking: thinking })
      },

      setThinkingTimeRemaining: (time) => {
        set({ thinkingTimeRemaining: time })
      },

      setSpeakingTimeRemaining: (time) => {
        set({ speakingTimeRemaining: time })
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

      setSessionEnded: (ended, reason) => {
        set({ sessionEnded: ended, sessionEndReason: reason || null })
      },
    }),
    {
      name: 'conversation-store',
    }
  )
)
