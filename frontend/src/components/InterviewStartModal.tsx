import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiService } from 'services/api'
import type { Agent } from 'services/api'
import { Button } from 'components/ui/Button'
import { Modal } from 'components/ui/Modal'
import { Avatar } from 'components/ui/Avatar'
import { useConversationStore } from 'store/useStore'

interface InterviewStartModalProps {
  isOpen: boolean
  onClose: () => void
  agent: Agent | null
  onStart?: () => void
}

export function InterviewStartModal({ isOpen, onClose, agent, onStart }: InterviewStartModalProps) {
  const navigate = useNavigate()
  const [isStarting, setIsStarting] = useState(false)

  const startSession = async () => {
    if (!agent) return
    
    try {
      setIsStarting(true)
      const response = await apiService.createSession(agent.id)
      useConversationStore.getState().setCurrentSession(response.session.id)
      
      if (onStart) {
        onStart()
      }
      
      onClose()
      navigate('/interview')
    } catch (err) {
      console.error('Failed to start session:', err)
      alert('Failed to start interview')
    } finally {
      setIsStarting(false)
    }
  }

  const handleClose = () => {
    if (!isStarting) {
      onClose()
    }
  }

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      title="Start Interview"
      size="md"
    >
      {agent && (
        <div className="space-y-6">
          <div className="text-center">
            <div className="flex justify-center mb-4">
              <Avatar 
                name={agent.name}
                role="ai"
                className="w-20 h-20"
              />
            </div>
            <h3 className="text-xl font-semibold mb-2">{agent.name}</h3>
            <p className="text-muted-foreground text-sm mb-4">{agent.description}</p>
            
            <div className="flex flex-wrap gap-2 justify-center mb-6">
              {agent.industry && (
                <span className="px-3 py-1 bg-orange-3 text-orange-11 text-sm rounded-full">
                  {agent.industry}
                </span>
              )}
              {agent.level && (
                <span className="px-3 py-1 bg-lime-3 text-lime-11 text-sm rounded-full">
                  {agent.level}
                </span>
              )}
              {agent.personality && (
                <span className="px-3 py-1 bg-blue-3 text-blue-11 text-sm rounded-full">
                  {agent.personality}
                </span>
              )}
            </div>
          </div>

          <div className="bg-muted p-4 rounded-lg">
            <h4 className="font-medium mb-2">Interview Information</h4>
            <ul className="text-sm space-y-1 text-muted-foreground">
              <li>• This will be a {agent.level?.toLowerCase() || 'general'} level interview</li>
              <li>• Focus on {agent.industry?.toLowerCase() || 'general'} skills and knowledge</li>
              <li>• The interview will be conducted by {agent.name}</li>
              <li>• Interviewer personality: {agent.personality || 'Professional'}</li>
              <li>• You can end the interview at any time</li>
            </ul>
          </div>

          <div className="flex space-x-3">
            <Button
              onClick={startSession}
              className="flex-1"
              disabled={isStarting}
            >
              {isStarting ? 'Starting...' : 'Start Interview'}
            </Button>
            <Button
              onClick={handleClose}
              variant="outline"
              className="flex-1"
              disabled={isStarting}
            >
              Cancel
            </Button>
          </div>
        </div>
      )}
    </Modal>
  )
}
