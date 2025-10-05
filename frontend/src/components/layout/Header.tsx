import { Link, useLocation } from 'react-router-dom'
import { Button } from 'components/ui/Button'
import { Avatar, AvatarFallback } from 'components/ui/Avatar'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from 'components/ui/DropdownMenu'
import { ThemeToggle } from 'components/ThemeToggle'
import { Logo } from 'components/ui/Logo'
import { useSignOut, useUser } from 'store/useAuth'

export function Header() {
  const user = useUser();
  const signOut = useSignOut();
  const location = useLocation()

  return (
    <header className="border-b bg-card backdrop-blur supports-[backdrop-filter]:bg-card">
      <div className="container flex h-14 items-center">
        <div className="mr-4 flex">
          <Link to="/" className="mr-6 flex items-center space-x-2">
            <Logo size="md" />
            <span className="hidden font-bold sm:inline-block">
              Praxis AI
            </span>
          </Link>
        </div>
        
        <div className="flex flex-1 items-center justify-between space-x-2">
          <div className="flex items-center space-x-2">
            <ThemeToggle />
          </div>
          <nav className="flex items-center space-x-4">
            {user && (
              <>
                <Link
                  to="/"
                  className={`text-sm font-medium transition-colors hover:text-primary ${
                    location.pathname === '/' ? 'text-primary' : 'text-muted-foreground'
                  }`}
                >
                  Dashboard
                </Link>
                <Link
                  to="/agents"
                  className={`text-sm font-medium transition-colors hover:text-primary ${
                    location.pathname === '/agents' ? 'text-primary' : 'text-muted-foreground'
                  }`}
                >
                  Agents
                </Link>
                <Link
                  to="/summaries"
                  className={`text-sm font-medium transition-colors hover:text-primary ${
                    location.pathname === '/summaries' ? 'text-primary' : 'text-muted-foreground'
                  }`}
                >
                  Summaries
                </Link>
              </>
            )}
            
            {user ? (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" className="relative h-8 w-8 rounded-full">
                    <Avatar className="h-8 w-8">
                      <AvatarFallback>
                        {user.email?.charAt(0).toUpperCase()}
                      </AvatarFallback>
                    </Avatar>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent className="w-56" align="end" forceMount>
                  <DropdownMenuLabel className="font-normal">
                    <div className="flex flex-col space-y-1">
                      <p className="text-sm font-medium leading-none">
                        {user.email}
                      </p>
                    </div>
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={signOut}>
                    Sign out
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            ) : (
              <Button variant="outline" size="sm">
                Sign In
              </Button>
            )}
          </nav>
        </div>
      </div>
    </header>
  )
}
