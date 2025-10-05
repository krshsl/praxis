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
  // Ref to control current audio element
  const audioRef = useRef<HTMLAudioElement | null>(null)

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
    isAudioPlaying,
    isTyping,
    typingContent,
    audioGenerationFailed,
    sessionEnded
  } = useConversationStore()
  const { toast } = useToast()
  // Show a full-page modal when the session ends (triggered by backend 'end_session')
  const [showSessionEndModal, setShowSessionEndModal] = useState(false)
  const [audioSrc, setAudioSrc] = useState<string | null>(null)
  const [currentPhase, setCurrentPhase] = useState<'thinking' | 'speaking' | 'idle'>('idle')
  const [warningCount, setWarningCount] = useState(0)

  const startThinkingPhase = useCallback(() => {
    setCurrentPhase('thinking')
    useConversationStore.getState().setThinking(true)
  }, [])

  const playAudio = useCallback(async (src: string) => {
    try {
      const audio = new Audio(src)
      audio.volume = 1.0
      audio.preload = 'auto'
      
      audio.addEventListener('play', () => {
        useConversationStore.getState().setAudioPlaying(true)
      })
      
      audio.addEventListener('ended', () => {
        setAudioSrc(null)
        useConversationStore.getState().setAudioPlaying(false)
        startThinkingPhase()
        audio.remove()
      })
      
      audio.addEventListener('error', () => {
        setAudioSrc(null)
        useConversationStore.getState().setAudioPlaying(false)
        startThinkingPhase()
        audio.remove()
      })
      
      audioRef.current = audio
      await audio.play()
      
    } catch (error) {
      setAudioSrc(null)
      useConversationStore.getState().setAudioPlaying(false)
      startThinkingPhase()
    }
  }, [startThinkingPhase])

  useEffect(() => {
    websocketService.setAudioCallback((audioSrc) => {
      setAudioSrc(audioSrc)
    })
    return () => {
      websocketService.setAudioCallback(() => {})
      if (audioRef.current) {
        audioRef.current.pause()
        audioRef.current.remove()
        audioRef.current = null
      }
    }
  }, [])

  useEffect(() => {
    if (audioSrc) {
      playAudio(audioSrc)
      
      const timeout = setTimeout(() => {
        setAudioSrc(null)
        useConversationStore.getState().setAudioPlaying(false)
        startThinkingPhase()
      }, 30000)
      
      return () => clearTimeout(timeout)
    }
  }, [audioSrc, playAudio, startThinkingPhase])

  useEffect(() => {
    if (sessionEnded) {
      // Show session end modal
      setShowSessionEndModal(true)
      
      // Clean up session state
      useConversationStore.getState().setCurrentSession(null)
      useConversationStore.getState().clearMessages()
      
      // Stop any playing audio
      if (audioRef.current) {
        audioRef.current.pause()
        audioRef.current.currentTime = 0
        audioRef.current.remove()
        audioRef.current = null
      }
      
      // Disconnect WebSocket to prevent further messages
      websocketService.disconnect()
      
      // Show appropriate toast based on session end reason
      const sessionEndReason = useConversationStore.getState().sessionEndReason
      if (sessionEndReason === 'Session ended by server') {
        toast({
          title: 'Session Ended Automatically',
          description: 'The interview session has been automatically ended. You will be redirected to the dashboard.',
          variant: 'default',
        })
      } else {
        toast({
          title: 'Interview Session Ended',
          description: 'Your interview session has ended. You will be redirected to the dashboard.',
          variant: 'default',
        })
      }
      
      // Reset session ended state and warning count
      setTimeout(() => {
        useConversationStore.getState().setSessionEnded(false)
        setWarningCount(0)
      }, 0)
    }
  }, [sessionEnded, toast])


  const [isInitialized, setIsInitialized] = useState(false)
  const [showStopModal, setShowStopModal] = useState(false)
  const [isEnding, setIsEnding] = useState(false)
  const lastAssistantMsgId = useRef<string | null>(null)
  

  useEffect(() => {
    if (!messages.length) return
    const last = messages[messages.length - 1]
    if (last.role === 'assistant') {
      if (lastAssistantMsgId.current === last.id) return
      lastAssistantMsgId.current = last.id
      if (!isProcessing && isConnected && currentPhase !== 'speaking' && !isAudioPlaying) {
        // Check if the last message has audio content (combined message)
        if (last.type === 'audio' && last.content) {
          const src = last.content
          if (src.startsWith('data:audio') || src.startsWith('http')) {
            setAudioSrc(src)
          } else {
            try {
              const byteString = atob(src)
              const ab = new ArrayBuffer(byteString.length)
              const ia = new Uint8Array(ab)
              for (let i = 0; i < byteString.length; i++) {
                ia[i] = byteString.charCodeAt(i)
              }
              const blob = new Blob([ab], { type: 'audio/mpeg' })
              const url = URL.createObjectURL(blob)
              setAudioSrc(url)
            } catch {
              // If audio parsing fails, start thinking immediately
              setTimeout(startThinkingPhase, 0)
            }
          }
        } else {
          // For text-only responses, start thinking immediately
          setTimeout(startThinkingPhase, 0)
        }
      }
    }
  }, [messages, isProcessing, isConnected, currentPhase, isAudioPlaying, startThinkingPhase])


  const initializeConnection = useCallback(async () => {
    try {
      if (user) {
        const currentSession = useConversationStore.getState().currentSession
        if (!currentSession) {
          navigate('/')
          return
        }
        await websocketService.connect()
      }
    } catch (error) {
      // Handle connection error silently
    }
  }, [navigate, user])

  useEffect(() => {
    if (user && !isInitialized) {
      initializeConnection()
      setWarningCount(0) // Reset warning count for new session
      setIsInitialized(true)
    }
  }, [user, isInitialized, initializeConnection])

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

  useEffect(() => {
    const handleStartThinkingPhase = () => {
      startThinkingPhase()
    }

    const interviewView = document.querySelector('[data-interview-view]')
    if (interviewView) {
      interviewView.addEventListener('startThinkingPhase', handleStartThinkingPhase)
      return () => {
        interviewView.removeEventListener('startThinkingPhase', handleStartThinkingPhase)
      }
    }
  }, [startThinkingPhase])

  useEffect(() => {
    if (audioGenerationFailed) {
      toast({
        title: 'Audio Generation Failed',
        description: 'Audio could not be generated. Please read the text response below. You will have extra time to read.',
        variant: 'default',
      })
    }
  }, [audioGenerationFailed, toast])

  // Monitor messages for warning indicators
  useEffect(() => {
    const latestMessage = messages[messages.length - 1]
    if (latestMessage && latestMessage.role === 'assistant') {
      const content = latestMessage.content.toLowerCase()
      
      // Check for warning patterns
      if (content.includes("couldn't hear") || content.includes("clear response") || content.includes("try again")) {
        setWarningCount(prev => prev + 1)
        
        // Show warning toast
        toast({
          title: 'Response Not Clear',
          description: 'Please speak clearly and provide a meaningful response.',
          variant: 'destructive',
        })
      } else if (content.includes("several attempts") || content.includes("end the session")) {
        // Final warning - session will end
        toast({
          title: 'Final Warning',
          description: 'The session will end if you continue to provide unclear responses.',
          variant: 'destructive',
        })
      }
    }
  }, [messages, toast])

  return (
    <div className="h-screen flex flex-col overflow-hidden" data-interview-view>
      {/* Session Ended Modal (full page) */}
      <Modal
        isOpen={showSessionEndModal}
        onClose={() => {}}
        title="Interview Ended"
        size="md"
        showCloseButton={false}
      >
        <div className="space-y-4 text-center">
          {useConversationStore.getState().sessionEndReason === 'Session ended by server' ? (
            <>
              <div className="w-16 h-16 mx-auto mb-4 bg-orange-100 dark:bg-orange-900/20 rounded-full flex items-center justify-center">
                <svg className="w-8 h-8 text-orange-600 dark:text-orange-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-foreground">Session Ended Automatically</h3>
              <p className="text-muted-foreground">
                The interview session has been automatically ended due to inactivity or uncooperative behavior. 
                Your responses will still be analyzed and a summary will be generated.
              </p>
            </>
          ) : (
            <>
              <div className="w-16 h-16 mx-auto mb-4 bg-green-100 dark:bg-green-900/20 rounded-full flex items-center justify-center">
                <svg className="w-8 h-8 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-foreground">Interview Completed</h3>
              <p className="text-muted-foreground">
                Your interview session has ended successfully. You will be redirected to the dashboard where you can view your summary.
              </p>
            </>
          )}
          
          <div className="bg-muted p-4 rounded-lg text-left">
            <h4 className="font-medium mb-2">What happens next?</h4>
            <ul className="text-sm text-muted-foreground space-y-1">
              <li>• Your interview responses will be analyzed by AI</li>
              <li>• A detailed summary and score will be generated</li>
              <li>• You can view your results on the dashboard</li>
              <li>• The summary will be available in a few minutes</li>
            </ul>
          </div>
          
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
              <div className="relative">
                <Avatar 
                  name={user?.email || "User"} 
                  role="user" 
                  isSpeaking={isUserSpeaking}
                  className="w-16 h-16"
                />
                {/* Warning indicator for uncooperative behavior */}
                {warningCount > 0 && (
                  <div className="absolute -top-1 -right-1 w-6 h-6 bg-orange-500 text-white rounded-full flex items-center justify-center text-xs font-bold">
                    {warningCount}
                  </div>
                )}
              </div>
              <SpeakingIndicator isSpeaking={isUserSpeaking} />
              {warningCount > 0 && (
                <div className="text-xs text-orange-600 dark:text-orange-400 text-center max-w-32">
                  {warningCount === 1 && "Please speak clearly"}
                  {warningCount === 2 && "Warning: Be more responsive"}
                  {warningCount >= 3 && "Final warning: Session may end"}
                </div>
              )}
              {/* Audio Playing Phase */}
              {isAudioPlaying && (
                <div className="text-center">
                  <p className="text-sm text-muted-foreground mb-2">AI is speaking...</p>
                  <div className="flex items-center justify-center space-x-2">
                    <div className="animate-pulse rounded-full h-4 w-4 bg-primary"></div>
                    <div className="animate-pulse rounded-full h-4 w-4 bg-primary" style={{ animationDelay: '0.2s' }}></div>
                    <div className="animate-pulse rounded-full h-4 w-4 bg-primary" style={{ animationDelay: '0.4s' }}></div>
                  </div>
                  <p className="text-xs text-muted-foreground mt-2">Please wait for the AI to finish speaking</p>
                </div>
              )}
              {/* Audio Generation Failed Phase */}
              {audioGenerationFailed && !isAudioPlaying && (
                <div className="text-center">
                  <div className="flex items-center justify-center mb-2">
                    <div className="w-4 h-4 bg-yellow-500 rounded-full mr-2"></div>
                    <p className="text-sm text-yellow-600 font-medium">Audio unavailable</p>
                  </div>
                  <p className="text-xs text-muted-foreground mb-2">
                    Please read the text response below
                  </p>
                  <div className="text-xs text-muted-foreground">
                    <p>⏱️ Taking time to read...</p>
                  </div>
                </div>
              )}
              {/* Thinking Phase */}
              {currentPhase === 'thinking' && !isAudioPlaying && (
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
            {messages.length === 0 && !isTyping ? (
              <div className="text-center text-muted-foreground py-8">
                <p>Connecting to interview... The AI will start the conversation automatically.</p>
              </div>
            ) : (
              <>
                {messages.map((message) => (
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
                ))}
                {/* Typing animation message */}
                {isTyping && (
                  <div className="flex justify-start mb-4">
                    <div className="max-w-[80%] rounded-lg px-4 py-3 bg-muted text-foreground border-l-4 border-primary">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="text-xs font-medium opacity-70">AI Interviewer</span>
                        <span className="text-xs opacity-50">typing...</span>
                      </div>
                      <p className="text-sm leading-relaxed">
                        {typingContent}
                        <span className="animate-pulse">|</span>
                      </p>
                    </div>
                  </div>
                )}
              </>
            )}
          </div>
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
