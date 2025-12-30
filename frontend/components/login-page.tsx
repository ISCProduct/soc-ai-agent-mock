'use client'

import { useState } from 'react'
import {
  Box,
  Card,
  CardContent,
  TextField,
  Button,
  Typography,
  Tabs,
  Tab,
  Divider,
  Alert,
  MenuItem,
} from '@mui/material'
import GitHubIcon from '@mui/icons-material/GitHub'
import GoogleIcon from '@mui/icons-material/Google'
import { authService, AuthResponse } from '@/lib/auth'

interface LoginPageProps {
  onAuthSuccess: (authResponse: AuthResponse) => void
}

export function LoginPage({ onAuthSuccess }: LoginPageProps) {
  const [tabValue, setTabValue] = useState(0)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [name, setName] = useState('')
  const [targetLevel, setTargetLevel] = useState('新卒')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const response = await authService.login(email, password)
      authService.saveAuth(response)
      onAuthSuccess(response)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const response = await authService.register(email, password, name, targetLevel)
      authService.saveAuth(response)
      onAuthSuccess(response)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleGuestLogin = async () => {
    setError('')
    setLoading(true)
    try {
      const response = await authService.createGuest()
      authService.saveAuth(response)
      onAuthSuccess(response)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleOAuthLogin = async (provider: 'google' | 'github') => {
    setError('')
    try {
      const response = provider === 'google' 
        ? await authService.getGoogleAuthUrl()
        : await authService.getGithubAuthUrl()
      
      localStorage.setItem('oauth_state', response.state)
      window.location.href = response.auth_url
    } catch (err: any) {
      setError(err.message)
    }
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
      <Card sx={{ maxWidth: 450, width: '100%' }}>
        <CardContent sx={{ p: 4 }}>
          <Typography variant="h4" align="center" gutterBottom fontWeight="bold">
            IT業界キャリアエージェント
          </Typography>
          <Typography variant="body2" align="center" color="text.secondary" sx={{ mb: 3 }}>
            4万社から最適な企業を選定
          </Typography>

          {error && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          )}

          <Tabs value={tabValue} onChange={(_, v) => setTabValue(v)} sx={{ mb: 3 }}>
            <Tab label="ログイン" />
            <Tab label="新規登録" />
          </Tabs>

          {tabValue === 0 ? (
            <Box component="form" onSubmit={handleLogin}>
              <TextField
                fullWidth
                label="メールアドレス"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                sx={{ mb: 2 }}
              />
              <TextField
                fullWidth
                label="パスワード"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                sx={{ mb: 3 }}
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
              <Button
                type="submit"
                fullWidth
                variant="contained"
                size="large"
                disabled={loading}
              >
                ログイン
              </Button>
            </Box>
          ) : (
            <Box component="form" onSubmit={handleRegister}>
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
                label="メールアドレス"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                sx={{ mb: 2 }}
              />
              <TextField
                fullWidth
                label="パスワード"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                sx={{ mb: 3 }}
              />
              <Button
                type="submit"
                fullWidth
                variant="contained"
                size="large"
                disabled={loading}
              >
                登録
              </Button>
            </Box>
          )}

          <Divider sx={{ my: 3 }}>または</Divider>

          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Button
              fullWidth
              variant="outlined"
              startIcon={<GoogleIcon />}
              onClick={() => handleOAuthLogin('google')}
              disabled={loading}
            >
              Googleでログイン
            </Button>
            <Button
              fullWidth
              variant="outlined"
              startIcon={<GitHubIcon />}
              onClick={() => handleOAuthLogin('github')}
              disabled={loading}
            >
              GitHubでログイン
            </Button>
            <Button
              fullWidth
              variant="text"
              onClick={handleGuestLogin}
              disabled={loading}
            >
              ゲストとして続ける
            </Button>
          </Box>
        </CardContent>
      </Card>
    </Box>
  )
}
