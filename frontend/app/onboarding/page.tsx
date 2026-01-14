'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Box, Button, Card, CardContent, MenuItem, TextField, Typography, Alert } from '@mui/material'
import { authService, User } from '@/lib/auth'
import { CERTIFICATION_OPTIONS, joinCertifications, splitCertifications } from '@/lib/profile'

export default function OnboardingPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [name, setName] = useState('')
  const [targetLevel, setTargetLevel] = useState('新卒')
  const [certificationsAcquired, setCertificationsAcquired] = useState<string[]>([])
  const [certificationsInProgress, setCertificationsInProgress] = useState('')
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
    setCertificationsAcquired(splitCertifications(storedUser.certifications_acquired))
    setCertificationsInProgress(storedUser.certifications_in_progress || '')
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
      const response = await authService.updateProfile(
        user.user_id,
        name,
        targetLevel,
        joinCertifications(certificationsAcquired),
        certificationsInProgress,
      )
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
            <TextField
              fullWidth
              select
              label="取得資格"
              value={certificationsAcquired}
              onChange={(e) =>
                setCertificationsAcquired(
                  typeof e.target.value === 'string' ? e.target.value.split(',') : e.target.value,
                )
              }
              SelectProps={{
                multiple: true,
                renderValue: (selected) => (selected as string[]).join(', '),
              }}
              helperText="複数選択可"
              sx={{ mb: 2 }}
            >
              {CERTIFICATION_OPTIONS.map((option) => (
                <MenuItem key={option} value={option}>
                  {option}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              fullWidth
              label="勉強中の資格"
              value={certificationsInProgress}
              onChange={(e) => setCertificationsInProgress(e.target.value)}
              placeholder="例: 応用情報技術者、AWS SAA（改行区切り可）"
              multiline
              minRows={3}
              sx={{ mb: 3 }}
            />
            <Button type="submit" fullWidth variant="contained" size="large" disabled={loading}>
              登録して診断を始める
            </Button>
          </Box>
        </CardContent>
      </Card>
    </Box>
  )
}
