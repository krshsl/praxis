import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiService } from 'services/api'
import type { Session, Summary } from 'services/api'
import { Button } from 'components/ui/Button'
import { SearchableTable } from 'components/ui/SearchableTable'
import { Modal } from 'components/ui/Modal'

export function SummaryPage() {
  const navigate = useNavigate()
  const [sessions, setSessions] = useState<Session[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedSessions, setSelectedSessions] = useState<Session[]>([])
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)
  const [deleteType, setDeleteType] = useState<'single' | 'bulk'>('single')
  const [sessionToDelete, setSessionToDelete] = useState<Session | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [summaries, setSummaries] = useState<Record<string, Summary | 'loading' | 'error'>>({})
  const [refreshing, setRefreshing] = useState(false)
  const summariesRef = useRef<Record<string, Summary | 'loading' | 'error'>>({})

  useEffect(() => {
    loadSessions()
  }, [])

  // Separate effect for periodic refresh
  useEffect(() => {
    const interval = setInterval(() => {
      const loadingSessions = sessions.filter(session => 
        session.status === 'completed' && summariesRef.current[session.id] === 'loading'
      )
      
      if (loadingSessions.length > 0) {
        loadSummariesForCompletedSessions(loadingSessions)
      }
    }, 5000) // Check every 5 seconds
    
    return () => clearInterval(interval)
  }, [sessions]) // Only depend on sessions, not summaries

  const loadSessions = async () => {
    try {
      setLoading(true)
      setError(null)
      const response = await apiService.getSessions()
      setSessions(response.sessions)
      
      // Load summaries for completed sessions
      await loadSummariesForCompletedSessions(response.sessions)
    } catch (err) {
      setError('Failed to load sessions')
      console.error('Sessions load error:', err)
    } finally {
      setLoading(false)
    }
  }

  const loadSummariesForCompletedSessions = async (sessions: Session[]) => {
    const completedSessions = sessions.filter(session => session.status === 'completed')
    
    for (const session of completedSessions) {
      try {
        setSummaries(prev => {
          const newSummaries = { ...prev, [session.id]: 'loading' as const }
          summariesRef.current = newSummaries
          return newSummaries
        })
        
        const summaryResponse = await apiService.getSummary(session.id)
        setSummaries(prev => {
          const updatedSummaries = { ...prev, [session.id]: summaryResponse.summary }
          summariesRef.current = updatedSummaries
          return updatedSummaries
        })
      } catch (err: any) {
        if (err.response?.status === 202) {
          // Summary is still being generated
          setSummaries(prev => {
            const updatedSummaries = { ...prev, [session.id]: 'loading' as const }
            summariesRef.current = updatedSummaries
            return updatedSummaries
          })
        } else {
          // Error or no summary available
          setSummaries(prev => {
            const updatedSummaries = { ...prev, [session.id]: 'error' as const }
            summariesRef.current = updatedSummaries
            return updatedSummaries
          })
        }
      }
    }
  }

  const refreshSummaries = async () => {
    setRefreshing(true)
    try {
      const completedSessions = sessions.filter(session => session.status === 'completed')
      await loadSummariesForCompletedSessions(completedSessions)
    } finally {
      setRefreshing(false)
    }
  }

  const viewSession = (sessionId: string) => {
    navigate(`/summary/${sessionId}`)
  }

  const handleDeleteSession = (session: Session) => {
    setSessionToDelete(session)
    setDeleteType('single')
    setDeleteModalOpen(true)
  }

  const handleBulkDelete = () => {
    if (selectedSessions.length === 0) return
    setDeleteType('bulk')
    setDeleteModalOpen(true)
  }

  const confirmDelete = async () => {
    if (!sessionToDelete && selectedSessions.length === 0) return

    try {
      setDeleting(true)
      
      if (deleteType === 'single' && sessionToDelete) {
        await apiService.deleteSession(sessionToDelete.id)
        setSessions(sessions.filter(s => s.id !== sessionToDelete.id))
      } else if (deleteType === 'bulk' && selectedSessions.length > 0) {
        const sessionIds = selectedSessions.map(s => s.id)
        await apiService.bulkDeleteSessions(sessionIds)
        setSessions(sessions.filter(s => !sessionIds.includes(s.id)))
        setSelectedSessions([])
      }
      
      setDeleteModalOpen(false)
      setSessionToDelete(null)
    } catch (err) {
      console.error('Delete error:', err)
      setError('Failed to delete session(s)')
    } finally {
      setDeleting(false)
    }
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
        // Only show score if session is completed
        if (session.status !== 'completed') {
          return (
            <div className="flex items-center space-x-2">
              <div className="w-12 h-12 relative">
                <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center">
                  <span className="text-xs text-muted-foreground">-</span>
                </div>
              </div>
              <div className="text-sm">
                <div className="font-medium text-muted-foreground">Not Available</div>
                <div className="text-muted-foreground">Session {session.status}</div>
              </div>
            </div>
          )
        }
        
        // For completed sessions, check summary status
        const summaryStatus = summaries[session.id]
        
        if (summaryStatus === 'loading') {
          return (
            <div className="flex items-center space-x-2">
              <div className="w-12 h-12 relative">
                <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center">
                  <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary"></div>
                </div>
              </div>
              <div className="text-sm">
                <div className="font-medium text-muted-foreground">Generating</div>
                <div className="text-muted-foreground">Summary in progress</div>
              </div>
            </div>
          )
        }
        
        if (summaryStatus === 'error') {
          return (
            <div className="flex items-center space-x-2">
              <div className="w-12 h-12 relative">
                <div className="w-12 h-12 rounded-full bg-destructive/10 flex items-center justify-center">
                  <span className="text-xs text-destructive">!</span>
                </div>
              </div>
              <div className="text-sm">
                <div className="font-medium text-muted-foreground">Error</div>
                <div className="text-muted-foreground">Summary unavailable</div>
              </div>
            </div>
          )
        }
        
        if (summaryStatus && typeof summaryStatus === 'object') {
          const score = summaryStatus.overall_score || 0
          const scoreColor = score >= 80 ? 'text-green-600' : score >= 60 ? 'text-yellow-600' : 'text-red-600'
          const bgColor = score >= 80 ? 'bg-green-100' : score >= 60 ? 'bg-yellow-100' : 'bg-red-100'
          
          return (
            <div className="flex items-center space-x-2">
              <div className="w-12 h-12 relative">
                <div className={`w-12 h-12 rounded-full ${bgColor} flex items-center justify-center`}>
                  <span className={`text-sm font-bold ${scoreColor}`}>{Math.round(score)} %</span>
                </div>
              </div>
              <div className="text-sm">
                <div className="text-muted-foreground">
                  {score >= 80 ? 'Excellent' : score >= 60 ? 'Good' : 'Needs Improvement'}
                </div>
              </div>
            </div>
          )
        }
        
        // Default case - no summary data yet
        return (
          <div className="flex items-center space-x-2">
            <div className="w-12 h-12 relative">
              <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center">
                <span className="text-xs text-muted-foreground">-</span>
              </div>
            </div>
            <div className="text-sm">
              <div className="font-medium text-muted-foreground">Pending</div>
              <div className="text-muted-foreground">No summary yet</div>
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
          <Button
            size="sm"
            variant="destructive"
            onClick={() => handleDeleteSession(session)}
          >
            Delete
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
        <div className="flex items-center space-x-4">
          <Button
            onClick={refreshSummaries}
            disabled={refreshing}
            variant="outline"
            size="sm"
          >
            {refreshing ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary mr-2"></div>
                Refreshing...
              </>
            ) : (
              'Refresh Scores'
            )}
          </Button>
          <div className="text-sm text-muted-foreground">
            {sessions.length} total sessions
          </div>
          {selectedSessions.length > 0 && (
            <Button
              variant="destructive"
              size="sm"
              onClick={handleBulkDelete}
            >
              Delete Selected ({selectedSessions.length})
            </Button>
          )}
        </div>
      </div>

      <SearchableTable
        data={sessions}
        columns={sessionColumns}
        searchFields={['status']}
        searchPlaceholder="Search by agent name or status..."
        emptyMessage="No interview sessions found. Start your first interview from the dashboard!"
        selectable={true}
        onSelectionChange={setSelectedSessions}
        getItemId={(session) => session.id}
      />

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteModalOpen}
        onClose={() => setDeleteModalOpen(false)}
        title={deleteType === 'single' ? 'Delete Session' : 'Delete Sessions'}
        size="sm"
      >
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">
            {deleteType === 'single' 
              ? `Are you sure you want to delete this interview session? This action cannot be undone.`
              : `Are you sure you want to delete ${selectedSessions.length} selected session(s)? This action cannot be undone.`
            }
          </p>
          
          {deleteType === 'single' && sessionToDelete && (
            <div className="p-3 bg-muted rounded-md">
              <p className="text-sm font-medium">{sessionToDelete.agent?.name || 'Unknown Agent'}</p>
              <p className="text-xs text-muted-foreground">
                Started: {new Date(sessionToDelete.started_at).toLocaleString()}
              </p>
            </div>
          )}

          {deleteType === 'bulk' && selectedSessions.length > 0 && (
            <div className="space-y-2">
              <p className="text-sm font-medium">Selected sessions:</p>
              <div className="max-h-32 overflow-y-auto space-y-1">
                {selectedSessions.map(session => (
                  <div key={session.id} className="p-2 bg-muted rounded text-xs">
                    {session.agent?.name || 'Unknown Agent'} - {new Date(session.started_at).toLocaleString()}
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="flex justify-end space-x-2">
            <Button
              variant="outline"
              onClick={() => setDeleteModalOpen(false)}
              disabled={deleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={confirmDelete}
              disabled={deleting}
            >
              {deleting ? 'Deleting...' : 'Delete'}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
