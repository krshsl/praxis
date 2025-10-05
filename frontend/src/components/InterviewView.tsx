import { useEffect, useState, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from 'components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from 'components/ui/Card'
import { Modal } from 'components/ui/Modal'
import { Avatar } from 'components/ui/Avatar'
import { SpeakingIndicator } from 'components/ui/SpeakingIndicator'
import { Timer } from 'components/ui/Timer'
import { useConversationStore } from 'store/useStore'
import { websocketService } from 'services/websocket'
import { audioService } from 'services/audio'
import { useUser } from 'store/useAuth'
import { INTERVIEW_TIMING } from 'constants/timing'
import { useToast } from 'hooks/useToast'

export function InterviewView() {
  const navigate = useNavigate()
  const user  = useUser()
  const { 
    messages, 
    isRecording, 
    isConnected, 
    audioLevel, 
    isProcessing,
    isUserSpeaking,
    isAISpeaking,
    sessionEnded
  } = useConversationStore()
  const { toast } = useToast()
  // Show a full-page modal when the session ends (triggered by backend 'end_session')
  const [showSessionEndModal, setShowSessionEndModal] = useState(false)
  useEffect(() => {
    if (sessionEnded) {
      setShowSessionEndModal(true)
      // Cleanup store, but do NOT navigate yet (wait for user to confirm)
      useConversationStore.getState().setCurrentSession(null)
      useConversationStore.getState().clearMessages()
      // Reset sessionEnded so effect can trigger again in future sessions
      setTimeout(() => {
        useConversationStore.getState().setSessionEnded(false)
      }, 0)
    }
  }, [sessionEnded])

  // Debug: Log messages when they change
  useEffect(() => {
    console.log('📝 Messages updated:', messages.length, 'messages')
    if (messages.length > 0) {
      console.log('Latest message:', messages[messages.length - 1])
    }
  }, [messages])

  const [isInitialized, setIsInitialized] = useState(false)
  const [showStopModal, setShowStopModal] = useState(false)
  const [isEnding, setIsEnding] = useState(false)
  const [currentPhase, setCurrentPhase] = useState<'thinking' | 'speaking' | 'idle'>('idle')
  const lastAssistantMsgId = useRef<string | null>(null)
  // ...existing code...
  

  // Start 10s thinking phase immediately after an AI question/message arrives
  useEffect(() => {
    if (!messages.length) return
    const last = messages[messages.length - 1]
    if (last.role === 'assistant') {
      if (lastAssistantMsgId.current === last.id) return
      lastAssistantMsgId.current = last.id
      if (!isProcessing && isConnected && currentPhase !== 'speaking') {
        startThinkingPhase()
      }
    }
  }, [messages, isProcessing, isConnected, currentPhase])

  // Timer logic is now handled by the Timer component itself

  const initializeConnection = useCallback(async () => {
    try {
      if (user) {
        const currentSession = useConversationStore.getState().currentSession
        if (!currentSession) {
          console.error('No session found. Please start a session from the dashboard first.')
          navigate('/')
          return
        }
        
        console.log('Initializing WebSocket connection for user:', user.email, 'with session:', currentSession)
        await websocketService.connect()
        console.log('WebSocket connection established')
      } else {
        console.error('No user found, cannot initialize WebSocket connection')
      }
    } catch (error) {
      console.error('Failed to initialize connection:', error)
    }
  }, [navigate, user])

  // Initialize websocket connection once per session when user is available
  useEffect(() => {
    if (user && !isInitialized) {
      initializeConnection()
      setIsInitialized(true)
    }
  }, [user, isInitialized, initializeConnection])

  const startThinkingPhase = () => {
    setCurrentPhase('thinking')
    useConversationStore.getState().setThinking(true)
  }

  const startSpeakingPhase = () => {
    setCurrentPhase('speaking')
    useConversationStore.getState().setThinking(false)
    
    // Start recording automatically
    audioService.startRecording()
    useConversationStore.getState().setRecording(true)
    useConversationStore.getState().setUserSpeaking(true)
  }

  const stopSpeakingPhase = async () => {
    setCurrentPhase('idle')
    useConversationStore.getState().setUserSpeaking(false)
    
    // Stop recording
    await audioService.stopRecording()
    useConversationStore.getState().setRecording(false)
  }

  // Silence detection during speaking to warn about time-wasting
  useEffect(() => {
    if (currentPhase !== 'speaking' || !isRecording) return

    let warned = false
    let silentMillis = 0
    const threshold = 0.06 // normalized audio level threshold
    const checkInterval = 250
    const maxSilentBeforeWarn = 10000 // 10s of silence

    const interval = setInterval(() => {
      const level = audioService.getAudioLevel()
      if (level < threshold) {
        silentMillis += checkInterval
        if (!warned && silentMillis >= maxSilentBeforeWarn) {
          warned = true
          toast({
            title: 'No audio detected',
            description: 'We’re not picking up your voice. Please speak up or move closer to the microphone.',
            variant: 'default'
          })
        }
      } else {
        // reset if we detect voice
        silentMillis = 0
      }
    }, checkInterval)

    return () => clearInterval(interval)
  }, [currentPhase, isRecording, toast])

  const handleManualStop = async () => {
    if (currentPhase === 'speaking') {
      await stopSpeakingPhase()
    }
  }

  // sending handled by audio/text flows, helper removed

  const handleEndInterview = async () => {
    setIsEnding(true)
    try {
      // End the session and generate summary
      await websocketService.sendMessage({
        type: 'end_session',
        session_id: useConversationStore.getState().currentSession || undefined
      })
      // Disconnect WebSocket first
      websocketService.disconnect()
      // Show toast confirming interview ended
      toast({
        title: 'Interview Ended',
        description: 'Your interview session has ended. You will be redirected to the dashboard.',
        variant: 'default',
      })
      // Clear the current session from store to allow new sessions
      useConversationStore.getState().setCurrentSession(null)
      // Clear conversation history
      useConversationStore.getState().clearMessages()
      console.log('Interview ended, session cleared, navigating to dashboard')
      navigate('/dashboard')
    } catch (error) {
      console.error('Failed to end interview:', error)
    } finally {
      setIsEnding(false)
      setShowStopModal(false)
    }
  }

  return (
    <div className="h-screen flex flex-col overflow-hidden">
      {/* Session Ended Modal (full page) */}
      <Modal
        isOpen={showSessionEndModal}
        onClose={() => {}}
        title="Interview Ended"
        size="md"
        showCloseButton={false}
      >
        <div className="space-y-4 text-center">
          <p className="text-lg text-muted-foreground">
            Your interview session has ended. You will be redirected to the dashboard.
          </p>
          <Button
            className="mt-4"
            onClick={() => {
              setShowSessionEndModal(false)
              navigate('/dashboard')
            }}
            autoFocus
          >
            Go to Dashboard
          </Button>
        </div>
      </Modal>
      {/* Header */}
  <div className="bg-card border-b border-border px-6 py-4 flex-shrink-0">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <h1 className="text-xl font-semibold">Live Interview</h1>
            <div className="flex items-center space-x-2">
              <div className={`w-3 h-3 rounded-full ${isConnected ? 'bg-primary' : 'bg-destructive'}`} />
              <span className="text-sm text-muted-foreground">
                {isConnected ? 'Connected' : 'Disconnected'}
              </span>
            </div>
          </div>
          <Button
            onClick={() => setShowStopModal(true)}
            variant="destructive"
            size="sm"
          >
            End Interview
          </Button>
        </div>
      </div>

  {/* Main Content - Two Columns */}
  <div className="flex-1 flex overflow-hidden max-h-[80vh]">
    {/* Left Column - AI Visual */}
    <div className="w-1/2 p-6 flex flex-col overflow-hidden">
      <Card className="flex-1 overflow-hidden">
        <CardHeader>
          <CardTitle>AI Interviewer</CardTitle>
        </CardHeader>
        <CardContent className="flex-1 flex flex-col items-center justify-center">
          {/* AI Avatar */}
          <div className="w-32 h-32 bg-primary text-primary-foreground rounded-full flex items-center justify-center text-4xl font-bold mb-6">
            AI
          </div>
          {/* Audio Controls */}
          <div className="space-y-4 w-full max-w-sm">
            {/* AI Avatar */}
            <div className="flex flex-col items-center space-y-2 mb-6">
              <Avatar 
                name="AI Interviewer" 
                role="ai" 
                isSpeaking={isAISpeaking}
                className="w-16 h-16"
              />
              <SpeakingIndicator isSpeaking={isAISpeaking} />
              {isProcessing && (
                <div className="flex items-center space-x-2">
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary"></div>
                  <span className="text-sm text-muted-foreground">Processing...</span>
                </div>
              )}
            </div>
            {/* User Avatar */}
            <div className="flex flex-col items-center space-y-2">
              <Avatar 
                name={user?.email || "User"} 
                role="user" 
                isSpeaking={isUserSpeaking}
                className="w-16 h-16"
              />
              <SpeakingIndicator isSpeaking={isUserSpeaking} />
              {/* Thinking Phase */}
              {currentPhase === 'thinking' && (
                <div className="text-center">
                  <p className="text-sm text-muted-foreground mb-2">Think about your response...</p>
                  <Timer
                    duration={INTERVIEW_TIMING.THINK_TIME}
                    onComplete={() => startSpeakingPhase()}
                    className="w-full mb-3"
                  />
                  <Button
                    onClick={startSpeakingPhase}
                    variant="outline"
                    size="sm"
                    className="w-full"
                  >
                    Start Speaking Now
                  </Button>
                </div>
              )}
              {/* Speaking Phase */}
              {currentPhase === 'speaking' && (
                <div className="text-center w-full">
                  <p className="text-sm text-muted-foreground mb-2">Speak now...</p>
                  <Timer
                    duration={INTERVIEW_TIMING.SPEAK_TIME}
                    onComplete={() => stopSpeakingPhase()}
                    className="w-full"
                  />
                  <Button
                    onClick={handleManualStop}
                    variant="outline"
                    size="sm"
                    className="mt-2"
                  >
                    Stop Early
                  </Button>
                </div>
              )}
            </div>
            {/* Audio Level Visualization */}
            {isRecording && (
              <div className="w-full">
                <div className="h-3 bg-muted rounded-full overflow-hidden">
                  <div
                    className="h-full bg-primary transition-all duration-100"
                    style={{ width: `${audioLevel * 100}%` }}
                  />
                </div>
                <p className="text-sm text-center text-muted-foreground mt-2">Audio Level</p>
              </div>
            )}
            {/* Processing Indicator */}
            {isProcessing && (
              <div className="w-full">
                <div className="flex items-center justify-center space-x-2">
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary"></div>
                  <p className="text-sm text-muted-foreground">Processing audio...</p>
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
    {/* Right Column - User Interface (Scrollable) */}
    <div className="w-1/2 p-6 flex flex-col overflow-hidden">
      <Card className="flex-1 flex flex-col overflow-hidden">
        <CardHeader className="flex-shrink-0">
          <CardTitle>Your Responses</CardTitle>
        </CardHeader>
        <CardContent className="flex-1 overflow-hidden p-6">
          {/* Messages */}
          <div className="h-full space-y-4 overflow-y-auto">
            {messages.length === 0 ? (
              <div className="text-center text-muted-foreground py-8">
                <p>Connecting to interview... The AI will start the conversation automatically.</p>
              </div>
            ) : (
              messages.map((message) => (
                <div
                  key={message.id}
                  className={`flex ${
                    message.role === 'user' ? 'justify-end' : 'justify-start'
                  } mb-4`}
                >
                  <div
                    className={`max-w-[80%] rounded-lg px-4 py-3 ${
                      message.role === 'user'
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted text-foreground border-l-4 border-primary'
                    }`}
                  >
                    <div className="flex items-center gap-2 mb-1">
                      <span className="text-xs font-medium opacity-70">
                        {message.role === 'user' ? 'You' : 'AI Interviewer'}
                      </span>
                      <span className="text-xs opacity-50">
                        {new Date(message.timestamp).toLocaleTimeString()}
                      </span>
                    </div>
                    <p className="text-sm leading-relaxed">{message.content}</p>
                  </div>
                </div>
              ))
            )}
          </div>
          {/* Quick Actions removed */}
        </CardContent>
      </Card>
    </div>
  </div>

      {/* Stop Interview Modal */}
      <Modal
        isOpen={showStopModal}
        onClose={() => setShowStopModal(false)}
        title="End Interview"
        size="sm"
      >
        <div className="space-y-4">
          <p className="text-muted-foreground">
            Are you sure you want to end this interview? The session will be saved and you can view the summary later.
          </p>
          <div className="flex space-x-3">
            <Button
              onClick={handleEndInterview}
              disabled={isEnding}
              variant="destructive"
              className="flex-1"
            >
              {isEnding ? 'Ending...' : 'End Interview'}
            </Button>
            <Button
              onClick={() => setShowStopModal(false)}
              variant="outline"
              className="flex-1"
            >
              Cancel
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
