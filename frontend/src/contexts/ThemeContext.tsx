import React, { useEffect, useState } from 'react'
import { ThemeContext, type Theme } from 'contexts/ThemeProvider'

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<Theme>('system')
  const [isLoaded, setIsLoaded] = useState(false)
  const [actualTheme, setActualTheme] = useState<'light' | 'dark'>(() => {
    if (typeof window !== 'undefined') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
    }
    return 'light'
  })

  useEffect(() => {
    const loadStoredTheme = () => {
      try {
        const storedTheme = localStorage.getItem('praxis-theme') as Theme
        if (storedTheme && ['light', 'dark', 'system'].includes(storedTheme)) {
          setTheme(storedTheme)
        } else {
          localStorage.setItem('praxis-theme', 'system')
        }
      } catch {
        // Fallback to system if localStorage fails
      }
      setIsLoaded(true)
    }

    loadStoredTheme()
  }, [])

  useEffect(() => {
    const updateActualTheme = () => {
      if (theme === 'system') {
        const systemPrefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
        setActualTheme(systemPrefersDark ? 'dark' : 'light')
      } else {
        setActualTheme(theme)
      }
    }

    updateActualTheme()

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    const handleChange = () => {
      if (theme === 'system') {
        updateActualTheme()
      }
    }

    mediaQuery.addEventListener('change', handleChange)
    return () => mediaQuery.removeEventListener('change', handleChange)
  }, [theme])

  useEffect(() => {
    const root = document.documentElement
    root.classList.remove('light', 'dark')
    root.classList.add(actualTheme)
  }, [actualTheme])

  useEffect(() => {
    if (isLoaded) {
      try {
        localStorage.setItem('praxis-theme', theme)
      } catch {
        // Ignore write errors
      }
    }
  }, [theme, isLoaded])

  const handleSetTheme = (newTheme: Theme) => {
    setTheme(newTheme)
  }

  return (
    <ThemeContext.Provider value={{ theme, setTheme: handleSetTheme, actualTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}
