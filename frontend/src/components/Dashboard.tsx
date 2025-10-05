import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiService } from 'services/api'
import type { Agent, Session } from 'services/api'
import { Card } from 'components/ui/Card'
import { Button } from 'components/ui/Button'
import { SearchableTable } from 'components/ui/SearchableTable'
import { Avatar } from 'components/ui/Avatar'
import { InterviewStartModal } from 'components/InterviewStartModal'
import { useConversationStore } from 'store/useStore'

export function Dashboard() {
  const navigate = useNavigate()
  const [agents, setAgents] = useState<Agent[]>([])
  const [sessions, setSessions] = useState<Session[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showInterviewModal, setShowInterviewModal] = useState(false)
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      setLoading(true)
      setError(null)
      
      const [agentsResponse, sessionsResponse] = await Promise.all([
        apiService.getAgents(),
        apiService.getSessions()
      ])
      
      // Filter for default agents only (agents without user_id)
      const defaultAgents = agentsResponse.agents.filter(agent => !agent.user_id)
      setAgents(defaultAgents)
      setSessions(sessionsResponse.sessions)
    } catch (err) {
      setError('Failed to load data')
      console.error('Dashboard load error:', err)
    } finally {
      setLoading(false)
    }
  }

  const showInterviewStartModal = (agent: Agent) => {
    setSelectedAgent(agent)
    setShowInterviewModal(true)
  }

  const handleModalClose = () => {
    setShowInterviewModal(false)
    setSelectedAgent(null)
  }

  const startCodingSession = async (agentId: string) => {
    try {
      const response = await apiService.createSession(agentId)
      useConversationStore.getState().setCurrentSession(response.session.id)
      navigate('/coding')
    } catch (err) {
      console.error('Failed to start coding session:', err)
      alert('Failed to start coding assessment')
    }
  }

  const viewSession = (sessionId: string) => {
    navigate(`/summary/${sessionId}`)
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-lg text-foreground">Loading...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <div className="text-destructive text-lg mb-4">{error}</div>
          <Button onClick={loadData}>Retry</Button>
        </div>
      </div>
    )
  }

  const sessionColumns: import('./ui/SearchableTable').Column<Session>[] = [
    {
      key: 'agent' as keyof Session,
      label: 'Agent',
      render: (_value: unknown, session: Session) => (
        <div className="font-medium">
          {session.agent?.name || 'Unknown Agent'}
        </div>
      )
    },
    {
      key: 'started_at' as keyof Session,
      label: 'Started',
  render: (value: unknown) => new Date((value as string) ?? '').toLocaleString(),
      sortable: true
    },
    {
      key: 'status' as keyof Session,
      label: 'Status',
      render: (value: unknown) => (
        <span className={`px-2 py-1 rounded text-sm border ${
          value === 'completed' 
            ? 'bg-muted text-lime-11 border-lime-8/30'
            : value === 'active'
            ? 'bg-muted text-orange-11 border-orange-8/30'
            : 'bg-muted text-gray-11 border-gray-8/30'
        }`}>
          {String(value ?? '')}
        </span>
      ),
      sortable: true
    },
    {
      key: 'id' as keyof Session,
      label: 'Actions',
      render: (_value: unknown, session: Session) => (
        <div className="flex space-x-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => viewSession(session.id)}
          >
            View
          </Button>
        </div>
      )
    }
  ]

  return (
    <div className="h-full flex flex-col">
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* AI Agents Section - Top 1/3 */}
        <div className="h-1/3 p-6 border-b border-border">
          <h1 className="text-2xl font-bold mb-4 text-foreground">Default AI Interviewers</h1>
          <div className="flex space-x-4 overflow-x-auto pb-2">
            {agents.length === 0 ? (
              <div className="flex-1 text-center py-8">
                <p className="text-muted-foreground">No default agents available</p>
              </div>
            ) : (
              agents.map((agent) => (
              <Card key={agent.id} className="min-w-[200px] p-4 flex-shrink-0 bg-card border-border">
                <div className="text-center">
                  <div className="flex justify-center mb-3">
                    <Avatar 
                      name={agent.name}
                      role="ai"
                      className="w-16 h-16"
                    />
                  </div>
                  <h3 className="font-semibold text-lg mb-2 text-card-foreground">{agent.name}</h3>
                  <p className="text-muted-foreground text-xs mb-3 line-clamp-3">{agent.description}</p>
                  <div className="flex flex-wrap gap-1 mb-4 justify-center">
                    {agent.industry && (
                      <span className="px-2 py-1 bg-orange-3 text-orange-11 text-xs rounded">
                        {agent.industry}
                      </span>
                    )}
                    {agent.level && (
                      <span className="px-2 py-1 bg-lime-3 text-lime-11 text-xs rounded">
                        {agent.level}
                      </span>
                    )}
                  </div>
                  <div className="space-y-2">
                    <Button 
                      onClick={() => showInterviewStartModal(agent)}
                      className="w-full"
                      size="sm"
                    >
                      Start Interview
                    </Button>
                    <Button 
                      onClick={() => startCodingSession(agent.id)}
                      className="w-full"
                      variant="outline"
                      size="sm"
                    >
                      Coding Test
                    </Button>
                  </div>
                </div>
              </Card>
              ))
            )}
          </div>
        </div>

        {/* Sessions Table - Bottom 2/3 */}
        <div className="flex-1 p-6 overflow-hidden">
          <h2 className="text-xl font-semibold mb-4 text-foreground">Interview History</h2>
          <SearchableTable
            data={sessions}
            columns={sessionColumns}
            searchFields={['status']}
            searchPlaceholder="Search by agent name or status..."
            emptyMessage="No interview sessions yet. Start your first interview above!"
            className="h-full"
          />
        </div>
      </div>

      {/* Interview Start Modal */}
      <InterviewStartModal
        isOpen={showInterviewModal}
        onClose={handleModalClose}
        agent={selectedAgent}
      />
    </div>
  )
}
