import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from 'components/ui/Card'
import { Button } from 'components/ui/Button'
import { CircularProgress } from 'components/ui/CircularProgress'

interface InterviewSummary {
  id: string
  agent: {
    name: string
    industry: string
  }
  started_at: string
  completed_at: string
  duration: number
  summary: string
  score: number
  strengths: string[]
  improvements: string[]
  technical_skills: {
    skill: string
    rating: number
  }[]
  communication_skills: {
    skill: string
    rating: number
  }[]
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
      
      // Mock data - in real app, you'd fetch from API
      const mockSummary: InterviewSummary = {
        id,
        agent: {
          name: 'Sarah Chen',
          industry: 'Software Engineering'
        },
        started_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(), // 2 hours ago
        completed_at: new Date().toISOString(),
        duration: 45, // minutes
        summary: `The candidate demonstrated strong problem-solving skills and clear communication throughout the interview. They showed good understanding of data structures and algorithms, particularly in the coding challenge where they implemented an efficient solution to the two-sum problem. The candidate asked thoughtful questions about the company culture and role expectations, showing genuine interest in the position. Areas for improvement include system design knowledge and handling edge cases in code. Overall, this was a solid performance that would be suitable for a mid-level developer position.`,
        score: 78,
        strengths: [
          'Clear communication and articulation',
          'Strong algorithmic thinking',
          'Good problem-solving approach',
          'Asks thoughtful questions',
          'Shows enthusiasm for the role'
        ],
        improvements: [
          'System design knowledge could be stronger',
          'Consider edge cases more thoroughly',
          'Practice explaining complex concepts',
          'Learn more about distributed systems'
        ],
        technical_skills: [
          { skill: 'Algorithms', rating: 85 },
          { skill: 'Data Structures', rating: 80 },
          { skill: 'System Design', rating: 60 },
          { skill: 'Code Quality', rating: 75 },
          { skill: 'Testing', rating: 70 }
        ],
        communication_skills: [
          { skill: 'Clarity', rating: 90 },
          { skill: 'Confidence', rating: 75 },
          { skill: 'Listening', rating: 85 },
          { skill: 'Questioning', rating: 80 }
        ]
      }
      
      setSummary(mockSummary)
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
            </CardContent>
          </Card>

          {/* Strengths and Improvements */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <Card>
              <CardHeader>
                <CardTitle className="text-lime-11">Strengths</CardTitle>
              </CardHeader>
              <CardContent>
                <ul className="space-y-2">
                  {summary.strengths.map((strength, index) => (
                    <li key={index} className="flex items-start space-x-2">
                      <span className="text-lime-9 mt-1">✓</span>
                      <span className="text-sm text-foreground">{strength}</span>
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-orange-11">Areas for Improvement</CardTitle>
              </CardHeader>
              <CardContent>
                <ul className="space-y-2">
                  {summary.improvements.map((improvement, index) => (
                    <li key={index} className="flex items-start space-x-2">
                      <span className="text-orange-9 mt-1">•</span>
                      <span className="text-sm text-foreground">{improvement}</span>
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
          </div>

        </div>

        {/* Score Section - 1/3 */}
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
                  <div className="text-2xl font-bold text-foreground">{summary.score}%</div>
                  <div className="text-sm text-muted-foreground">
                    {summary.score >= 80 ? 'Excellent' : 
                     summary.score >= 70 ? 'Good' : 
                     summary.score >= 60 ? 'Fair' : 'Needs Improvement'}
                  </div>
                </div>

                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-foreground">Technical Skills</span>
                    <span className="font-medium text-foreground">
                      {Math.round(summary.technical_skills.reduce((acc, skill) => acc + skill.rating, 0) / summary.technical_skills.length)}%
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-foreground">Communication</span>
                    <span className="font-medium text-foreground">
                      {Math.round(summary.communication_skills.reduce((acc, skill) => acc + skill.rating, 0) / summary.communication_skills.length)}%
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
                  {summary.technical_skills.map((skill, index) => (
                    <div key={index}>
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
                  ))}
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Communication Skills</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {summary.communication_skills.map((skill, index) => (
                    <div key={index}>
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
                  ))}
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  )
}
