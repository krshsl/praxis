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
    console.log('📨 WebSocket message received:', data)

    if (data.type === 'end_session') {
      // Handle session end from backend
      store.setCurrentSession(null)
      store.clearMessages()
      store.setSessionEnded(true, 'Session ended by server')
      // Optionally disconnect WebSocket here if needed
      this.disconnect()
      return
    }

    if (data.type === 'audio' && ('audio_data' in data || 'audio_data_base64' in data)) {
      // TODO: Re-enable audio handling later
      console.log('🎵 Audio message received (disabled for now)')
      // const audioData = (data as any).audio_data || (data as any).audio_data_base64
      // this.playAudio(audioData)
    } else if ('content' in data) {
      // Handle text/code message
      console.log('💬 Text message received:', data.content)
      // Determine role based on message type
      const role = data.type === 'user_message' ? 'user' : 'assistant'
      console.log('🔍 Message type:', data.type, 'Role:', role, 'Content:', data.content)
      store.addMessage({
        content: data.content || '',
        role: role,
        type: data.type as 'text' | 'code' | 'audio',
        language: data.language,
      })
      store.setProcessing(false)
    } else {
      console.log('❓ Unknown message type:', data)
    }
  }

  // Audio playback method removed (disabled for now)

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
    } else {
      console.error('WebSocket is not connected, state:', this.ws?.readyState)
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
        
        console.log(`🎵 Sending audio chunk ${chunkIndex + 1}/${totalChunks}: ${audioData.length} characters`)
        
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
    } else {
      console.error('❌ WebSocket is not connected, state:', this.ws?.readyState)
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
    console.log('WebSocket disconnected and reset')
  }

  getConnectionState(): number {
    return this.ws?.readyState ?? WebSocket.CLOSED
  }
}

export const websocketService = new WebSocketService(
  import.meta.env.VITE_WS_URL || 'ws://localhost:8080/api/v1/ws'
)
