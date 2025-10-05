import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import { MainLayout } from 'components/layout/MainLayout'
import { LoginForm } from 'components/auth/LoginForm'
import { SignUpForm } from 'components/auth/SignUpForm'
import { Dashboard } from 'components/Dashboard'
import { InterviewView } from 'components/InterviewView'
import { CodingView } from 'components/CodingView'
import { AgentsPage } from 'components/AgentsPage'
import { SummaryPage } from 'components/SummaryPage'
import { InterviewSummaryPage } from 'components/InterviewSummaryPage'
import { useUser, useAuthLoading, useIsAuthChecked, useCheckAuth } from 'store/useAuth'
import { Toaster } from 'components/ui/Toaster'
import { ThemeProvider } from 'contexts/ThemeContext'
import { ProtectedRoute } from 'components/auth/ProtectedRoute'
import BackgroundCanvas from 'components/layout/BackgroundCanvas'
import { useEffect } from 'react'
import './theme.css'

function App() {
  const user = useUser()
  const isAuthChecked = useIsAuthChecked()
  const loading = useAuthLoading()
  const checkAuth = useCheckAuth()

  useEffect(() => {
    if (!isAuthChecked) {
      checkAuth()
    }
  }, [checkAuth, isAuthChecked])

  if (!isAuthChecked || loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
        <div className="ml-3 text-sm text-muted-foreground">Loading...</div>
      </div>
    )
  }

  return (
    <ThemeProvider>
      <BackgroundCanvas />
      <Router>
        <Routes>
          <Route path="/login" element={user ? <Navigate to="/" replace /> : <LoginForm />} />
          <Route path="/signup" element={user ? <Navigate to="/" replace /> : <SignUpForm />} />

          <Route element={<ProtectedRoute />}>
            <Route element={<MainLayout />}>
              <Route index element={<Dashboard />} />
              <Route path="interview" element={<InterviewView />} />
              <Route path="coding" element={<CodingView />} />
              <Route path="agents" element={<AgentsPage />} />
              <Route path="summaries" element={<SummaryPage />} />
              <Route path="summary/:sessionId" element={<InterviewSummaryPage />} />
            </Route>
          </Route>
          
          <Route path="*" element={<Navigate to={user ? "/" : "/login"} replace />} />
        </Routes>
        <Toaster />
      </Router>
    </ThemeProvider>
  )
}

export default App;
