import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from 'components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from 'components/ui/Card'
import { Input } from 'components/ui/Input'
import { Label } from 'components/ui/Label'
import { useLogin } from 'store/useAuth'

export function LoginForm() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [formData, setFormData] = useState({
    email: '',
    password: ''
  })
  
  const login = useLogin()
  const navigate = useNavigate()

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

    try {
      await login(formData.email, formData.password)
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
          <CardTitle>Welcome to Praxis</CardTitle>
          <CardDescription>
            AI-powered interview practice platform
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                name="email"
                type="email"
                required
                value={formData.email}
                onChange={handleInputChange}
                placeholder="Enter your email"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                name="password"
                type="password"
                required
                value={formData.password}
                onChange={handleInputChange}
                placeholder="Enter your password"
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
              {loading ? 'Signing in...' : 'Sign In'}
            </Button>

            <div className="text-center text-sm text-muted-foreground">
              <p>
                Don't have an account?{' '}
                <button 
                  type="button"
                  className="text-primary hover:underline"
                  onClick={() => navigate('/signup')}
                >
                  Sign up
                </button>
              </p>
            </div>

          </form>
        </CardContent>
      </Card>
    </div>
  )
}
