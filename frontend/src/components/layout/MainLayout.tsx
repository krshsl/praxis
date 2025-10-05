import { Outlet } from 'react-router-dom'
import { Header } from 'components/layout/Header'

export function MainLayout() {
  return (
    <div className="min-h-screen flex flex-col">
      <Header />
      <main className="flex-1 overflow-hidden">
        <Outlet />
      </main>
    </div>
  )
}
