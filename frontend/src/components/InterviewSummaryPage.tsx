import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from 'components/ui/Card'
import { Button } from 'components/ui/Button'
import { CircularProgress } from 'components/ui/CircularProgress'
import { apiService } from 'services/api'

interface InterviewSummary {
  id: string
  agent: {
    name: string
    industry: string
    personality: string
    level: string
  }
  started_at: string
  completed_at: string
  duration: number
  summary: string
  score: number
  strengths: string
  weaknesses: string
  recommendations: string
  technical_skills: {
    skill: string
    rating: number
  }[]
  communication_skills: {
    skill: string
    rating: number
  }[]
  isGenerating?: boolean
}

export function InterviewSummaryPage() {
  const { sessionId } = useParams<{ sessionId: string }>()
  const navigate = useNavigate()
  const [summary, setSummary] = useState<InterviewSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (sessionId) {
      loadSummary(sessionId)
    }
  }, [sessionId])

  const loadSummary = async (id: string) => {
    try {
      setLoading(true)
      setError(null)
      
      // First get the session data
      const sessionResponse = await apiService.getSession(id)
      const session = sessionResponse.session
      
      // Try to get the summary
      let summary = null
      try {
        const summaryResponse = await apiService.getSummary(id)
        summary = summaryResponse.summary
      } catch (summaryError: any) {
        // If summary is still being generated (202), show loading state
        if (summaryError.response?.status === 202) {
          setSummary({
            id: session.id,
            agent: {
              name: session.agent?.name || 'Unknown',
              industry: session.agent?.industry || 'Unknown',
              personality: session.agent?.personality || 'Unknown',
              level: session.agent?.level || 'Unknown'
            },
            started_at: session.started_at,
            completed_at: session.ended_at || new Date().toISOString(),
            duration: Math.round(session.duration / 60),
            summary: 'Summary is being generated. Please wait...',
            score: 0,
            strengths: 'Analysis in progress...',
            weaknesses: 'Analysis in progress...',
            recommendations: 'Analysis in progress...',
            technical_skills: [],
            communication_skills: [],
            isGenerating: true
          })
          setLoading(false)
          return
        } else if (summaryError.response?.status === 404) {
          console.log('Summary not found for session:', id)
        } else {
          throw summaryError
        }
      }
      
      const interviewSummary: InterviewSummary = {
        id: session.id,
        agent: {
          name: session.agent?.name || 'Unknown',
          industry: session.agent?.industry || 'Unknown',
          personality: session.agent?.personality || 'Unknown',
          level: session.agent?.level || 'Unknown'
        },
        started_at: session.started_at,
        completed_at: session.ended_at || new Date().toISOString(),
        duration: Math.round(session.duration / 60),
        summary: summary?.summary || 'Summary is being generated. Please check back in a few minutes.',
        score: summary?.overall_score || 0,
        strengths: summary?.strengths || 'Analysis in progress...',
        weaknesses: summary?.weaknesses || 'Analysis in progress...',
        recommendations: summary?.recommendations || 'Analysis in progress...',
        technical_skills: [],
        communication_skills: [],
        isGenerating: !summary
      }
      
      setSummary(interviewSummary)
    } catch (err) {
      setError('Failed to load interview summary')
      console.error('Summary load error:', err)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-lg">Loading interview summary...</div>
      </div>
    )
  }

  // Show generating state when summary is being created
  if (summary?.isGenerating) {
    return (
      <div className="min-h-screen bg-background">
        <div className="container mx-auto px-4 py-8">
          <div className="max-w-4xl mx-auto">
            <div className="text-center mb-8">
              <h1 className="text-3xl font-bold mb-2">Interview Summary</h1>
              <p className="text-muted-foreground">Your interview analysis is being generated</p>
            </div>

            <Card className="p-8 text-center">
              <div className="space-y-6">
                <div className="flex justify-center">
                  <div className="animate-spin rounded-full h-16 w-16 border-b-2 border-primary"></div>
                </div>
                
                <div>
                  <h2 className="text-2xl font-semibold mb-2">AI is analyzing your interview</h2>
                  <p className="text-muted-foreground mb-4">
                    Our AI is reviewing your responses and generating a personalized summary with detailed feedback.
                  </p>
                </div>

                <div className="bg-muted p-4 rounded-lg">
                  <h3 className="font-medium mb-2">What's happening?</h3>
                  <ul className="text-sm text-muted-foreground space-y-1 text-left">
                    <li>• Analyzing your responses and communication style</li>
                    <li>• Evaluating technical knowledge and problem-solving skills</li>
                    <li>• Generating personalized feedback and recommendations</li>
                    <li>• Calculating performance scores</li>
                  </ul>
                </div>

                <div className="flex justify-center space-x-4">
                  <Button
                    onClick={() => sessionId && loadSummary(sessionId)}
                    className="bg-primary text-primary-foreground"
                  >
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                    Check Status
                  </Button>
                  <Button
                    onClick={() => navigate('/dashboard')}
                    variant="outline"
                  >
                    Back to Dashboard
                  </Button>
                </div>

                <p className="text-xs text-muted-foreground">
                  This usually takes 1-2 minutes. You can check back later or we'll notify you when it's ready.
                </p>
              </div>
            </Card>
          </div>
        </div>
      </div>
    )
  }

  if (error || !summary) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="text-destructive text-lg mb-4">{error || 'Summary not found'}</div>
          <Button onClick={() => navigate('/dashboard')}>Back to Dashboard</Button>
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold">Interview Summary</h1>
          <p className="text-muted-foreground mt-2">
            Interview with {summary.agent.name} • {summary.agent.industry}
          </p>
        </div>
        <Button onClick={() => navigate('/dashboard')} variant="outline">
          Back to Dashboard
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Summary Content - 2/3 */}
        <div className="lg:col-span-2 space-y-6">
          {/* Overview */}
          <Card>
            <CardHeader>
              <CardTitle>Interview Overview</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
                <div className="text-center">
                  <div className="text-2xl font-bold text-primary">{summary.duration} min</div>
                  <div className="text-sm text-muted-foreground">Duration</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-foreground">
                    {new Date(summary.started_at).toLocaleDateString()}
                  </div>
                  <div className="text-sm text-muted-foreground">Date</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-accent-foreground">
                    {summary.agent.name}
                  </div>
                  <div className="text-sm text-muted-foreground">Interviewer</div>
                  <div className="text-xs text-muted-foreground mt-1">
                    {summary.agent.personality} • {summary.agent.level}
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Summary Text */}
          <Card>
            <CardHeader>
              <CardTitle>Interview Summary</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-foreground leading-relaxed">{summary.summary}</p>
              {summary.summary.includes('being generated') && (
                <div className="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-2">
                      <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600"></div>
                      <span className="text-sm text-blue-700">AI is analyzing your interview and generating a personalized summary...</span>
                    </div>
                    <Button
                      onClick={() => sessionId && loadSummary(sessionId)}
                      size="sm"
                      variant="outline"
                      className="text-blue-700 border-blue-300 hover:bg-blue-100"
                    >
                      Refresh
                    </Button>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Strengths and Improvements */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <Card>
              <CardHeader>
                <CardTitle className="text-lime-11">Strengths</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-foreground leading-relaxed">{summary.strengths}</p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-orange-11">Areas for Improvement</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-foreground leading-relaxed">{summary.weaknesses}</p>
              </CardContent>
            </Card>
          </div>

          {/* Recommendations */}
          <Card>
            <CardHeader>
              <CardTitle className="text-blue-11">Recommendations</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-foreground leading-relaxed">{summary.recommendations}</p>
            </CardContent>
          </Card>

        </div>

        {/* Score Section - 1/3 - Only show when summary is ready */}
        {!summary.isGenerating && (
          <div className="lg:col-span-1">
            <Card className="sticky top-6">
              <CardHeader>
                <CardTitle>Overall Score</CardTitle>
              </CardHeader>
              <CardContent className="text-center">
                <CircularProgress
                  value={summary.score}
                  size={150}
                  label="Overall Performance"
                  showPercentage={true}
                  className="mx-auto mb-6"
                />
                
                <div className="space-y-4">
                  <div className="text-center">
                    <div className="text-2xl font-bold text-foreground">
                      {summary.score}%
                    </div>
                    <div className="text-sm text-muted-foreground">
                      {summary.score >= 80 ? 'Excellent' : 
                       summary.score >= 70 ? 'Good' : 
                       summary.score >= 60 ? 'Fair' : 'Needs Improvement'}
                    </div>
                    <div className="text-xs text-muted-foreground mt-2 p-2 bg-muted rounded">
                      Scored by {summary.agent.name} ({summary.agent.personality} interviewer)
                    </div>
                  </div>

                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-foreground">Technical Skills</span>
                      <span className="font-medium text-foreground">
                        {summary.technical_skills.length > 0 
                          ? Math.round(summary.technical_skills.reduce((acc, skill) => acc + skill.rating, 0) / summary.technical_skills.length)
                          : 'N/A'
                        }%
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-foreground">Communication</span>
                      <span className="font-medium text-foreground">
                        {summary.communication_skills.length > 0 
                          ? Math.round(summary.communication_skills.reduce((acc, skill) => acc + skill.rating, 0) / summary.communication_skills.length)
                          : 'N/A'
                        }%
                      </span>
                    </div>
                  </div>

                  <Button 
                    onClick={() => navigate('/dashboard')} 
                    className="w-full"
                  >
                    Back to Dashboard
                  </Button>
                </div>
              </CardContent>
            </Card>

            {/* Skills Assessment */}
            <div className="space-y-6 mt-6">
              <Card>
                <CardHeader>
                  <CardTitle>Technical Skills</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    {summary.technical_skills.length > 0 ? (
                      summary.technical_skills.map((skill, index) => (
                        <div key={`tech-${skill.skill}-${index}`}>
                          <div className="flex justify-between items-center mb-1">
                            <span className="text-sm font-medium text-foreground">{skill.skill}</span>
                            <span className="text-sm text-muted-foreground">{skill.rating}%</span>
                          </div>
                          <div className="w-full bg-muted rounded-full h-2">
                            <div
                              className="bg-primary h-2 rounded-full transition-all duration-300"
                              style={{ width: `${skill.rating}%` }}
                            />
                          </div>
                        </div>
                      ))
                    ) : (
                      <p className="text-sm text-muted-foreground">No technical skills data available</p>
                    )}
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Communication Skills</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    {summary.communication_skills.length > 0 ? (
                      summary.communication_skills.map((skill, index) => (
                        <div key={`comm-${skill.skill}-${index}`}>
                          <div className="flex justify-between items-center mb-1">
                            <span className="text-sm font-medium text-foreground">{skill.skill}</span>
                            <span className="text-sm text-muted-foreground">{skill.rating}%</span>
                          </div>
                          <div className="w-full bg-muted rounded-full h-2">
                            <div
                              className="bg-primary h-2 rounded-full transition-all duration-300"
                              style={{ width: `${skill.rating}%` }}
                            />
                          </div>
                        </div>
                      ))
                    ) : (
                      <p className="text-sm text-muted-foreground">No communication skills data available</p>
                    )}
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
