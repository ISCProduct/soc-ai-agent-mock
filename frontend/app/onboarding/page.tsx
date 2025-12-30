'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Box, Button, Card, CardContent, MenuItem, TextField, Typography, Alert } from '@mui/material'
import { authService, User } from '@/lib/auth'

export default function OnboardingPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [name, setName] = useState('')
  const [targetLevel, setTargetLevel] = useState('新卒')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    const storedUser = authService.getStoredUser()
    if (!storedUser) {
      router.replace('/login')
      return
    }
    setUser(storedUser)
    setName(storedUser.name || '')
    if (storedUser.target_level === '新卒' || storedUser.target_level === '中途') {
      setTargetLevel(storedUser.target_level)
    }
  }, [router])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!user) return

    setError('')
    setLoading(true)
    try {
      const response = await authService.updateProfile(user.user_id, name, targetLevel)
      authService.saveAuth(response)
      router.replace('/')
    } catch (err: any) {
      setError(err.message || 'Failed to update profile')
    } finally {
      setLoading(false)
    }
  }

  if (!user) {
    return null
  }

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        bgcolor: 'background.default',
        p: 2,
      }}
    >
      <Card sx={{ maxWidth: 480, width: '100%' }}>
        <CardContent sx={{ p: 4 }}>
          <Typography variant="h5" gutterBottom fontWeight="bold">
            まずは簡単な情報を教えてください
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
            診断の質問内容を最適化するために必要です。
          </Typography>

          {error && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          )}

          <Box component="form" onSubmit={handleSubmit}>
            <TextField
              fullWidth
              label="名前"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              sx={{ mb: 2 }}
            />
            <TextField
              fullWidth
              select
              label="区分"
              value={targetLevel}
              onChange={(e) => setTargetLevel(e.target.value)}
              sx={{ mb: 3 }}
            >
              <MenuItem value="新卒">新卒</MenuItem>
              <MenuItem value="中途">中途</MenuItem>
            </TextField>
            <Button type="submit" fullWidth variant="contained" size="large" disabled={loading}>
              登録して診断を始める
            </Button>
          </Box>
        </CardContent>
      </Card>
    </Box>
  )
}
