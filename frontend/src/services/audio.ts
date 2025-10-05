import { websocketService } from 'services/websocket'
import { useConversationStore } from 'store/useStore'

class AudioService {
  private mediaRecorder: MediaRecorder | null = null
  private audioChunks: Blob[] = []
  private audioContext: AudioContext | null = null
  private analyser: AnalyserNode | null = null
  private microphone: MediaStreamAudioSourceNode | null = null
  private animationFrame: number | null = null

  async startRecording(): Promise<void> {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ 
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          sampleRate: 44100,
        } 
      })

      this.mediaRecorder = new MediaRecorder(stream, {
        mimeType: 'audio/webm;codecs=opus'
      })

      this.audioChunks = []

      this.mediaRecorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          this.audioChunks.push(event.data)
        }
      }

      this.mediaRecorder.onstop = () => {
        const audioBlob = new Blob(this.audioChunks, { type: 'audio/webm' })
        const sizeMB = (audioBlob.size / 1024 / 1024).toFixed(2)
        console.log('🎤 Audio recording stopped, sending audio blob:', audioBlob.size, 'bytes', `(${sizeMB} MB)`)
        
        // Send audio in chunks if it's too large
        this.sendAudioInChunks(audioBlob)
        this.audioChunks = []
      }

      this.mediaRecorder.start(100) // Collect data every 100ms
      useConversationStore.getState().setRecording(true)

      // Set up audio visualization
      this.setupAudioVisualization(stream)

    } catch (error) {
      console.error('Error starting recording:', error)
      throw error
    }
  }

  stopRecording(): void {
    if (this.mediaRecorder && this.mediaRecorder.state === 'recording') {
      this.mediaRecorder.stop()
      useConversationStore.getState().setRecording(false)
    }

    // Stop audio visualization
    if (this.animationFrame) {
      cancelAnimationFrame(this.animationFrame)
      this.animationFrame = null
    }

    if (this.audioContext) {
      this.audioContext.close()
      this.audioContext = null
    }
  }

  private setupAudioVisualization(stream: MediaStream): void {
    try {
      this.audioContext = new AudioContext()
      this.analyser = this.audioContext.createAnalyser()
      this.microphone = this.audioContext.createMediaStreamSource(stream)

      this.analyser.fftSize = 256
      this.microphone.connect(this.analyser)

      this.visualizeAudio()
    } catch (error) {
      console.error('Error setting up audio visualization:', error)
    }
  }

  private visualizeAudio(): void {
    if (!this.analyser) return

    const dataArray = new Uint8Array(this.analyser.frequencyBinCount)
    
    const updateLevel = () => {
      this.analyser!.getByteFrequencyData(dataArray)
      
      // Calculate average volume
      const average = dataArray.reduce((sum, value) => sum + value, 0) / dataArray.length
      const normalizedLevel = average / 255
      
      useConversationStore.getState().setAudioLevel(normalizedLevel)
      
      this.animationFrame = requestAnimationFrame(updateLevel)
    }

    updateLevel()
  }

  async playAudio(audioBlob: Blob): Promise<void> {
    try {
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

  isRecording(): boolean {
    return this.mediaRecorder?.state === 'recording' || false
  }

  getAudioLevel(): number {
    return useConversationStore.getState().audioLevel
  }

  private async sendAudioInChunks(audioBlob: Blob) {
    const chunkSize = 2 * 1024 * 1024 // 2MB chunks
    const totalSize = audioBlob.size
    
    if (totalSize <= chunkSize) {
      // Small enough to send directly
      websocketService.sendAudio(audioBlob)
      return
    }

    console.log(`📦 Splitting audio into chunks: ${totalSize} bytes -> ${Math.ceil(totalSize / chunkSize)} chunks`)
    
    const chunks: Blob[] = []
    let offset = 0
    
    while (offset < totalSize) {
      const end = Math.min(offset + chunkSize, totalSize)
      const chunk = audioBlob.slice(offset, end)
      chunks.push(chunk)
      offset = end
    }

    // Send chunks sequentially with a small delay
    for (let i = 0; i < chunks.length; i++) {
      const chunk = chunks[i]
      const isLastChunk = i === chunks.length - 1
      
      console.log(`📤 Sending chunk ${i + 1}/${chunks.length}: ${chunk.size} bytes${isLastChunk ? ' (final)' : ''}`)
      
      // Send chunk with metadata
      websocketService.sendAudioChunk(chunk, i, chunks.length, isLastChunk)
      
      // Small delay between chunks to prevent overwhelming the server
      if (!isLastChunk) {
        await new Promise(resolve => setTimeout(resolve, 100))
      }
    }
  }
}

export const audioService = new AudioService()
