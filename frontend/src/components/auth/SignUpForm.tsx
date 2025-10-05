import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from 'components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from 'components/ui/Card'
import { useSignUp } from 'store/useAuth'
import { useToast } from 'hooks/useToast'

export function SignUpForm() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    password: '',
    confirmPassword: ''
  })
  
  const signUp = useSignUp()
  const navigate = useNavigate()
  const { toast } = useToast()

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    setFormData(prev => ({
      ...prev,
      [name]: value
    }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)

    if (formData.password !== formData.confirmPassword) {
      setError('Passwords do not match')
      setLoading(false)
      return
    }

    try {
      await signUp(formData.name, formData.email, formData.password)
      
      // Show success toast
      toast({
        title: "Account Created Successfully!",
        description: "Your account has been created. Please sign in to continue.",
        variant: "default"
      })
      
      // Redirect to login page
      navigate('/signin')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Create Account</CardTitle>
          <CardDescription>
            Join Praxis and start practicing with AI interviewers
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="name" className="block text-sm font-medium mb-1">
                Full Name
              </label>
              <input
                id="name"
                name="name"
                type="text"
                required
                value={formData.name}
                onChange={handleInputChange}
                className="w-full px-3 py-2 border border-input rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
                placeholder="Enter your full name"
              />
            </div>

            <div>
              <label htmlFor="email" className="block text-sm font-medium mb-1">
                Email
              </label>
              <input
                id="email"
                name="email"
                type="email"
                required
                value={formData.email}
                onChange={handleInputChange}
                className="w-full px-3 py-2 border border-input rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
                placeholder="Enter your email"
              />
            </div>

            <div>
              <label htmlFor="password" className="block text-sm font-medium mb-1">
                Password
              </label>
              <input
                id="password"
                name="password"
                type="password"
                required
                value={formData.password}
                onChange={handleInputChange}
                className="w-full px-3 py-2 border border-input rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
                placeholder="Create a password"
              />
            </div>

            <div>
              <label htmlFor="confirmPassword" className="block text-sm font-medium mb-1">
                Confirm Password
              </label>
              <input
                id="confirmPassword"
                name="confirmPassword"
                type="password"
                required
                value={formData.confirmPassword}
                onChange={handleInputChange}
                className="w-full px-3 py-2 border border-input rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
                placeholder="Confirm your password"
              />
            </div>

            {error && (
              <div className="text-sm text-destructive">
                {error}
              </div>
            )}

            <Button 
              type="submit"
              className="w-full" 
              disabled={loading}
            >
              {loading ? 'Creating Account...' : 'Create Account'}
            </Button>

            <div className="text-center text-sm text-muted-foreground">
              <p>
                Already have an account?{' '}
                <button 
                  type="button"
                  className="text-primary hover:underline"
                  onClick={() => navigate('/signin')}
                >
                  Sign in
                </button>
              </p>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
