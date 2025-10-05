import { useConversationStore } from 'store/useStore'

export interface WebSocketMessage {
  type: 'text' | 'code' | 'audio' | 'end_session' | 'user_message'
  content?: string
  language?: string
  session_id?: string
}

export interface AudioMessage {
  type: 'audio'
  audio_data: string
  session_id: string
}

export interface CombinedAudioMessage {
  type: 'audio'
  content?: string
  language?: string
  session_id?: string
  audio_data?: string
  audio_data_base64?: string
  AudioDataBase64?: string
}

class WebSocketService {
  private ws: WebSocket | null = null
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000
  private isConnecting = false

  constructor(url: string) {
    this.url = url
  }
  
  private url: string

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.isConnecting || (this.ws && this.ws.readyState === WebSocket.OPEN)) {
        resolve()
        return
      }

      this.isConnecting = true

      try {
        // Get the current session ID from the store
        const currentSession = useConversationStore.getState().currentSession
        if (!currentSession) {
          reject(new Error('No session ID available. Please start a session first.'))
          return
        }

        // Build WebSocket URL with session ID parameter
        const wsUrl = `${this.url}?session_id=${currentSession}`
        this.ws = new WebSocket(wsUrl)

        this.ws.onopen = () => {
          console.log('WebSocket connected successfully')
          this.isConnecting = false
          this.reconnectAttempts = 0
          useConversationStore.getState().setConnected(true)
          resolve()
        }

        this.ws.onmessage = (event) => {
          try {
            // Support multiple JSON objects in one message (newline-delimited)
            const messages = event.data
              .split(/\r?\n/)
              .map((line: string) => line.trim())
              .filter((line: string) => line.length > 0)
            for (const msg of messages) {
              try {
                const data = JSON.parse(msg)
                this.handleMessage(data)
              } catch (err) {
                console.error('Error parsing WebSocket message chunk:', err, msg)
              }
            }
          } catch (error) {
            console.error('Error processing WebSocket message:', error)
          }
        }

        this.ws.onclose = (event) => {
          console.log('WebSocket disconnected:', event.code, event.reason)
          this.isConnecting = false
          useConversationStore.getState().setConnected(false)
          
          // Handle different close codes
          if (event.code === 1006) {
            console.error('WebSocket connection lost unexpectedly')
          } else if (event.code === 1002) {
            console.error('WebSocket protocol error')
          } else if (event.code === 1003) {
            console.error('WebSocket unsupported data type')
          } else if (event.code === 1000) {
            console.log('WebSocket closed normally')
          } else if (event.code === 1001) {
            console.log('WebSocket going away')
          }
          
          if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
            console.log(`Attempting to reconnect (${this.reconnectAttempts + 1}/${this.maxReconnectAttempts})`)
            this.scheduleReconnect()
          } else if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.error('Max reconnection attempts reached')
          }
        }

        this.ws.onerror = (error) => {
          console.error('WebSocket connection error:', error)
          this.isConnecting = false
          useConversationStore.getState().setConnected(false)
          reject(new Error(`WebSocket connection failed: ${error}`))
        }

      } catch (error) {
        this.isConnecting = false
        reject(error)
      }
    })
  }

  private scheduleReconnect() {
    this.reconnectAttempts++
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1)
    
    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`)
    
    setTimeout(() => {
      this.connect()
    }, delay)
  }

  private handleMessage(data: WebSocketMessage | AudioMessage) {
    const store = useConversationStore.getState()

    if (data.type === 'end_session') {
      store.setCurrentSession(null)
      store.clearMessages()
      store.setSessionEnded(true, 'Session ended by server')
      this.disconnect()
      return
    }

    if (data.type === 'audio') {
      const audioMessage = data as CombinedAudioMessage
      const audioData = audioMessage.audio_data || audioMessage.audio_data_base64 || audioMessage.AudioDataBase64
      
      if (audioData) {
        store.setAudioGenerationFailed(false)
        this.playAudio(audioData)
      } else {
        store.setAudioGenerationFailed(true)
      }
      
      if ('content' in data && data.content) {
        store.setTyping(true)
        store.setTypingContent(data.content)
        this.startTypingAnimation(data.content, !audioData)
      }
      store.setProcessing(false)
    } else if ('content' in data) {
      const role = data.type === 'user_message' ? 'user' : 'assistant'
      store.addMessage({
        content: data.content || '',
        role: role,
        type: data.type as 'text' | 'code' | 'audio',
        language: data.language,
      })
      store.setProcessing(false)
    }
  }



  setAudioCallback(callback: (audioSrc: string) => void) {
    this._audioCallback = callback
  }
  private _audioCallback?: (audioSrc: string) => void

  private startTypingAnimation(content: string, audioFailed: boolean = false) {
    const store = useConversationStore.getState()
    let currentIndex = 0
    const typingSpeed = 30 // milliseconds per character
    
    const typeNextChar = () => {
      if (currentIndex < content.length) {
        const partialContent = content.substring(0, currentIndex + 1)
        store.setTypingContent(partialContent)
        currentIndex++
        setTimeout(typeNextChar, typingSpeed)
      } else {
        // Typing complete, add the full message
        store.setTyping(false)
        store.addMessage({
          content: content,
          role: 'assistant',
          type: 'audio',
          language: undefined,
        })
        
        if (audioFailed) {
          const wordCount = content.split(' ').length
          const readingTimeMs = Math.max(5000, (wordCount / 200) * 60 * 1000)
          
          setTimeout(() => {
            const interviewView = document.querySelector('[data-interview-view]')
            if (interviewView) {
              interviewView.dispatchEvent(new CustomEvent('startThinkingPhase'))
            }
          }, readingTimeMs)
        }
      }
    }
    
    setTimeout(typeNextChar, typingSpeed)
  }

  private playAudio(audioData: string) {
    try {
      let audioSrc: string
      
      if (audioData.startsWith('data:audio')) {
        audioSrc = audioData
      } else {
        const byteString = atob(audioData)
        const ab = new ArrayBuffer(byteString.length)
        const ia = new Uint8Array(ab)
        for (let i = 0; i < byteString.length; i++) {
          ia[i] = byteString.charCodeAt(i)
        }
        const blob = new Blob([ab], { type: 'audio/mpeg' })
        audioSrc = URL.createObjectURL(blob)
      }
      
      if (this._audioCallback) {
        this._audioCallback(audioSrc)
      }
    } catch (e) {
      // Handle audio processing error silently
    }
  }


  sendMessage(message: WebSocketMessage) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message))
      
      if (message.type !== 'end_session') {
        useConversationStore.getState().addMessage({
          content: message.content || '',
          role: 'user',
          type: message.type as 'text' | 'code' | 'audio',
          language: message.language,
        })
      }
    }
  }

  sendAudio(audioBlob: Blob) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      useConversationStore.getState().setProcessing(true)
      
      const reader = new FileReader()
      reader.onload = () => {
        const arrayBuffer = reader.result as ArrayBuffer
        const uint8Array = new Uint8Array(arrayBuffer)
        const audioData = btoa(String.fromCharCode(...uint8Array))
        
        this.ws?.send(JSON.stringify({
          type: 'audio',
          audio_data: audioData,
          session_id: useConversationStore.getState().currentSession
        }))
      }
      reader.readAsArrayBuffer(audioBlob)
    }
  }

  sendAudioChunk(audioBlob: Blob, chunkIndex: number, totalChunks: number, isLastChunk: boolean) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      if (chunkIndex === 0) {
        useConversationStore.getState().setProcessing(true)
      }
      const reader = new FileReader()
      reader.onload = () => {
        const arrayBuffer = reader.result as ArrayBuffer
        const uint8Array = new Uint8Array(arrayBuffer)
        const audioData = btoa(String.fromCharCode(...uint8Array))
        
        
        this.ws?.send(JSON.stringify({
          type: 'audio_chunk',
          audio_data: audioData,
          chunk_index: chunkIndex,
          total_chunks: totalChunks,
          is_last_chunk: isLastChunk,
          session_id: useConversationStore.getState().currentSession
        }))
      }
      reader.readAsArrayBuffer(audioBlob)
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.isConnecting = false
    this.reconnectAttempts = 0
    useConversationStore.getState().setConnected(false)
  }

  getConnectionState(): number {
    return this.ws?.readyState ?? WebSocket.CLOSED
  }
}

export const websocketService = new WebSocketService(
  import.meta.env.VITE_WS_URL || 'ws://localhost:8080/api/v1/ws'
)
