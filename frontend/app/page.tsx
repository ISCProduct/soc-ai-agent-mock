'use client'

import { useState, useEffect } from 'react'
import { Box } from '@mui/material'
import { AnalysisSidebar } from '@/components/analysis-sidebar'
import { MuiChat } from '@/components/mui-chat'
import { LoginPage } from '@/components/login-page'
import { authService, User, AuthResponse } from '@/lib/auth'

export default function Home() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const storedUser = authService.getStoredUser()
    setUser(storedUser)
    setLoading(false)
  }, [])

  const handleAuthSuccess = (authResponse: AuthResponse) => {
    // Normalize types from backend
    setUser({
      ...authResponse,
      user_id: Number((authResponse as any).user_id),
    })
  }

  const handleLogout = () => {
    authService.logout()
    setUser(null)
  }

  if (loading) {
    return null
  }

  if (!user) {
    return <LoginPage onAuthSuccess={handleAuthSuccess} />
  }

  return (
    <Box sx={{ display: 'flex', height: '100vh', overflow: 'hidden' }}>
      <AnalysisSidebar user={user} onLogout={handleLogout} />
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          height: '100vh',
        }}
      >
        <MuiChat />
      </Box>
    </Box>
  )
}

