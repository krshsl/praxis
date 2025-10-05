import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from 'components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from 'components/ui/Card'
import { Modal } from 'components/ui/Modal'
import { useConversationStore } from 'store/useStore'
import { websocketService } from 'services/websocket'
import { audioService } from 'services/audio'
import { useUser } from 'store/useAuth'

export function InterviewView() {
  const navigate = useNavigate()
  const user  = useUser()
  const { 
    messages, 
    isRecording, 
    isConnected, 
    audioLevel, 
    isProcessing
  } = useConversationStore()

  const [isInitialized, setIsInitialized] = useState(false)
  const [showStopModal, setShowStopModal] = useState(false)
  const [isEnding, setIsEnding] = useState(false)

  useEffect(() => {
    if (user && !isInitialized) {
      initializeConnection()
      setIsInitialized(true)
    }
  }, [user, isInitialized])

  const initializeConnection = async () => {
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
  }

  const handleStartRecording = async () => {
    try {
      await audioService.startRecording()
    } catch (error) {
      console.error('Failed to start recording:', error)
    }
  }

  const handleStopRecording = () => {
    audioService.stopRecording()
  }

  const handleSendMessage = (content: string) => {
    if (isConnected) {
      websocketService.sendMessage({
        type: 'text',
        content,
        session_id: useConversationStore.getState().currentSession || undefined
      })
    }
  }

  const handleEndInterview = async () => {
    setIsEnding(true)
    try {
      // End the session and generate summary
      await websocketService.sendMessage({
        type: 'end_session',
        session_id: useConversationStore.getState().currentSession || undefined
      })
      navigate('/dashboard')
    } catch (error) {
      console.error('Failed to end interview:', error)
    } finally {
      setIsEnding(false)
      setShowStopModal(false)
    }
  }

  return (
  <div className="h-screen flex flex-col">
      {/* Header */}
  <div className="bg-card border-b border-border px-6 py-4">
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
      <div className="flex-1 flex">
        {/* Left Column - AI Visual */}
        <div className="w-1/2 p-6 flex flex-col">
          <Card className="flex-1">
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
                <Button
                  onClick={isRecording ? handleStopRecording : handleStartRecording}
                  disabled={!isConnected || isProcessing}
                  variant={isRecording ? 'destructive' : 'default'}
                  size="lg"
                  className="w-full"
                >
                  {isRecording ? 'Stop Recording' : 'Start Recording'}
                </Button>

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
                  <div className="flex items-center justify-center space-x-2 text-sm text-muted-foreground">
                    <div className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin" />
                    <span>AI is thinking...</span>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Right Column - User Interface */}
        <div className="w-1/2 p-6 flex flex-col">
          <Card className="flex-1">
            <CardHeader>
              <CardTitle>Your Responses</CardTitle>
            </CardHeader>
            <CardContent className="flex-1 flex flex-col">
              {/* Messages */}
              <div className="flex-1 space-y-4 overflow-y-auto mb-4">
                {messages.length === 0 ? (
                  <div className="text-center text-muted-foreground py-8">
                    Start the interview by clicking "Start Recording"
                  </div>
                ) : (
                  messages.map((message) => (
                    <div
                      key={message.id}
                      className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
                    >
                      <div
                        className={`max-w-xs px-4 py-2 rounded-lg ${
                          message.role === 'user'
                            ? 'bg-primary text-primary-foreground'
                            : 'bg-muted text-foreground'
                        }`}
                      >
                        <p className="text-sm">{message.content}</p>
                        <p className="text-xs opacity-70 mt-1">
                          {new Date(message.timestamp).toLocaleTimeString()}
                        </p>
                      </div>
                    </div>
                  ))
                )}
              </div>

              {/* Quick Actions */}
              <div className="space-y-2">
                <p className="text-sm font-medium text-foreground">Quick Actions:</p>
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleSendMessage("Hello, I'm ready to start the interview")}
                    disabled={!isConnected}
                  >
                    Start Interview
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleSendMessage("Can you repeat that question?")}
                    disabled={!isConnected}
                  >
                    Repeat Question
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleSendMessage("I need a moment to think")}
                    disabled={!isConnected}
                  >
                    Thinking Time
                  </Button>
                </div>
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
