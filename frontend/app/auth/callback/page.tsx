'use client'

import { useEffect, useState, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { Box, CircularProgress, Typography, Alert } from '@mui/material'

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:80'

function OAuthCallbackContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [error, setError] = useState('')

  useEffect(() => {
    const handleCallback = async () => {
      const errorParam = searchParams.get('error')
      if (errorParam) {
        setError(decodeURIComponent(errorParam))
        return
      }

      const userParam = searchParams.get('user')
      const provider = searchParams.get('provider')
      
      if (!userParam) {
        setError('ユーザー情報が見つかりません')
        return
      }

      try {
        // Base64デコードしてユーザー情報を取得
        const userDataString = atob(userParam)
        const userData = JSON.parse(userDataString)
        
        // ローカルストレージに保存
        localStorage.setItem('user', JSON.stringify(userData))
        if (userData.token) {
          localStorage.setItem('token', userData.token)
        }
        localStorage.removeItem('oauth_state')

        // ホームページにリダイレクト
        router.push('/')
      } catch (err: any) {
        setError('認証データの処理に失敗しました: ' + err.message)
      }
    }

    handleCallback()
  }, [searchParams, router])

  if (error) {
    return (
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          minHeight: '100vh',
          p: 3,
        }}
      >
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
        <Typography
          variant="body2"
          color="primary"
          sx={{ cursor: 'pointer' }}
          onClick={() => router.push('/')}
        >
          ホームに戻る
        </Typography>
      </Box>
    )
  }

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        gap: 2,
      }}
    >
      <CircularProgress />
      <Typography variant="body1">認証中...</Typography>
    </Box>
  )
}

export default function OAuthCallback() {
  return (
    <Suspense fallback={
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          minHeight: '100vh',
          gap: 2,
        }}
      >
        <CircularProgress />
        <Typography variant="body1">読み込み中...</Typography>
      </Box>
    }>
      <OAuthCallbackContent />
    </Suspense>
  )
}
