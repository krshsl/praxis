import { useConversationStore } from 'store/useStore'

export interface WebSocketMessage {
  type: 'text' | 'code' | 'audio' | 'end_session'
  content?: string
  language?: string
  session_id?: string
}

export interface AudioMessage {
  type: 'audio'
  audio_data: string
  session_id: string
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
            const data = JSON.parse(event.data)
            this.handleMessage(data)
          } catch (error) {
            console.error('Error parsing WebSocket message:', error)
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

    if (data.type === 'audio' && 'audio_data' in data) {
      // Handle audio message
      this.playAudio(data.audio_data)
    } else if ('content' in data) {
      // Handle text/code message
      store.addMessage({
        content: data.content || '',
        role: 'assistant',
        type: data.type as 'text' | 'code' | 'audio',
        language: data.language,
      })
    }
  }

  private async playAudio(audioData: string) {
    try {
      const audioBlob = new Blob([Uint8Array.from(atob(audioData), c => c.charCodeAt(0))], {
        type: 'audio/mpeg'
      })
      
      const audioUrl = URL.createObjectURL(audioBlob)
      const audio = new Audio(audioUrl)
      
      audio.onended = () => {
        URL.revokeObjectURL(audioUrl)
      }
      
      await audio.play()
    } catch (error) {
      console.error('Error playing audio:', error)
    }
  }

  sendMessage(message: WebSocketMessage) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message))
      
      // Add user message to store (only for text/code/audio messages)
      if (message.type !== 'end_session') {
        useConversationStore.getState().addMessage({
          content: message.content || '',
          role: 'user',
          type: message.type as 'text' | 'code' | 'audio',
          language: message.language,
        })
      }
    } else {
      console.error('WebSocket is not connected')
    }
  }

  sendAudio(audioBlob: Blob) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
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
    } else {
      console.error('WebSocket is not connected')
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    useConversationStore.getState().setConnected(false)
  }

  getConnectionState(): number {
    return this.ws?.readyState ?? WebSocket.CLOSED
  }
}

export const websocketService = new WebSocketService(
  import.meta.env.VITE_WS_URL || 'ws://localhost:8080/api/v1/ws'
)
