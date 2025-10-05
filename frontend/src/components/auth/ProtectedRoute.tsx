import { Navigate, Outlet } from 'react-router-dom'
import { useUser, useAuthLoading, useIsAuthChecked } from 'store/useAuth'

export const ProtectedRoute = () => {
  const user = useUser()
  const loading = useAuthLoading()
  const isAuthChecked = useIsAuthChecked()

  if (loading || !isAuthChecked) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
        <div className="ml-3 text-sm text-muted-foreground">Loading...</div>
      </div>
    )
  }

  if (!user) {
    return <Navigate to="/login" replace />
  }

  return <Outlet />
}
