import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiService } from 'services/api'
import type { Session } from 'services/api'
import { Button } from 'components/ui/Button'
import { SearchableTable } from 'components/ui/SearchableTable'

export function SummaryPage() {
  const navigate = useNavigate()
  const [sessions, setSessions] = useState<Session[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadSessions()
  }, [])

  const loadSessions = async () => {
    try {
      setLoading(true)
      setError(null)
      const response = await apiService.getSessions()
      setSessions(response.sessions)
    } catch (err) {
      setError('Failed to load sessions')
      console.error('Sessions load error:', err)
    } finally {
      setLoading(false)
    }
  }

  const viewSession = (sessionId: string) => {
    navigate(`/summary/${sessionId}`)
  }

  const sessionColumns = [
    {
      key: 'agent' as keyof Session,
      label: 'Agent',
      render: (_value: any, session: Session) => (
        <div className="flex items-center space-x-3">
          <div className="w-10 h-10 bg-primary text-primary-foreground rounded-full flex items-center justify-center font-bold text-sm">
            {session.agent?.name?.charAt(0) || 'A'}
          </div>
          <div>
            <div className="font-medium">{session.agent?.name || 'Unknown Agent'}</div>
            <div className="text-sm text-muted-foreground">{session.agent?.industry || ''}</div>
          </div>
        </div>
      )
    },
    {
      key: 'started_at' as keyof Session,
      label: 'Started',
      render: (value: any) => new Date(value).toLocaleString(),
      sortable: true
    },
    {
      key: 'status' as keyof Session,
      label: 'Status',
      render: (value: any) => (
        <span className={`px-2 py-1 rounded text-sm ${
          value === 'completed' 
            ? 'bg-accent text-accent-foreground'
            : value === 'active'
            ? 'bg-orange-3 text-orange-11'
            : 'bg-muted text-foreground'
        }`}>
          {value}
        </span>
      ),
      sortable: true
    },
    {
      key: 'id' as keyof Session,
      label: 'Score',
      render: (_value: any, session: Session) => {
        // Calculate score based on session data - in real app this would come from the backend
        const score = session.duration > 30 ? 85 : session.duration > 15 ? 75 : 65
        return (
          <div className="flex items-center space-x-2">
            <div className="w-12 h-12 relative">
              <svg className="w-12 h-12 transform -rotate-90">
                <circle
                  cx="24"
                  cy="24"
                  r="20"
                  stroke="#e5e7eb"
                  strokeWidth="4"
                  fill="none"
                />
                <circle
                  cx="24"
                  cy="24"
                  r="20"
                  stroke={score >= 80 ? '#10b981' : score >= 60 ? '#f59e0b' : '#ef4444'}
                  strokeWidth="4"
                  fill="none"
                  strokeDasharray={`${2 * Math.PI * 20}`}
                  strokeDashoffset={`${2 * Math.PI * 20 * (1 - score / 100)}`}
                  strokeLinecap="round"
                />
              </svg>
              <div className="absolute inset-0 flex items-center justify-center">
                <span className="text-xs font-bold">{score}%</span>
              </div>
            </div>
          </div>
        )
      }
    },
    {
      key: 'id' as keyof Session,
      label: 'Actions',
      render: (_value: any, session: Session) => (
        <div className="flex space-x-2">
          <Button
            size="sm"
            onClick={() => viewSession(session.id)}
          >
            View Details
          </Button>
        </div>
      )
    }
  ]

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-lg">Loading sessions...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="text-destructive text-lg mb-4">{error}</div>
          <Button onClick={loadSessions}>Retry</Button>
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-3xl font-bold">Interview Summaries</h1>
  <div className="text-sm text-muted-foreground">
          {sessions.length} total sessions
        </div>
      </div>

      <SearchableTable
        data={sessions}
        columns={sessionColumns}
            searchFields={['status']}
        searchPlaceholder="Search by agent name or status..."
        emptyMessage="No interview sessions found. Start your first interview from the dashboard!"
      />
    </div>
  )
}
