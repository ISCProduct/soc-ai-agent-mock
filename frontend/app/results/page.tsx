'use client'

import { useState, useEffect, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import {
  Box,
  Paper,
  Typography,
  Button,
  Card,
  CardContent,
  Stack,
  CircularProgress,
  Avatar,
  Chip,
} from '@mui/material'
import { ArrowBack, LocationOn, People, TrendingUp as TrendingUpIcon, Refresh } from '@mui/icons-material'

interface Company {
  id: string
  name: string
  industry: string
  location: string
  employees: string
  description: string
  matchScore: number
  tags: string[]
  techStack: string[]
}

function ResultsContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [companies, setCompanies] = useState<Company[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [mounted, setMounted] = useState(false)

  const userId = searchParams.get('user_id')
  const sessionId = searchParams.get('session_id')

  useEffect(() => {
    setMounted(true)
  }, [])

  useEffect(() => {
    if (!mounted || !userId || !sessionId) {
      if (mounted && (!userId || !sessionId)) {
        setError('セッション情報が見つかりません')
        setLoading(false)
      }
      return
    }

    const fetchCompanies = async () => {
      try {
        setLoading(true)
        const response = await fetch(`/api/chat/recommendations?user_id=${userId}&session_id=${sessionId}&limit=10`)
        
        if (!response.ok) {
          throw new Error('企業データの取得に失敗しました')
        }
        
        const data = await response.json()
        
        if (data && data.recommendations && Array.isArray(data.recommendations)) {
          const mappedCompanies = data.recommendations.map((rec: any, index: number) => ({
            id: String(rec.id || index + 1),
            name: rec.category_name || `企業 ${index + 1}`,
            industry: rec.category_name || 'IT',
            location: '東京都',
            employees: '未定',
            description: rec.reason || '詳細情報は準備中です',
            matchScore: rec.score || 0,
            tags: [],
            techStack: [],
          }))
          setCompanies(mappedCompanies)
        } else {
          setError('企業データの形式が正しくありません')
        }
      } catch (err) {
        console.error('企業データ取得エラー:', err)
        setError(err instanceof Error ? err.message : '不明なエラー')
      } finally {
        setLoading(false)
      }
    }

    fetchCompanies()
  }, [mounted, userId, sessionId])

  const handleBack = () => {
    router.push('/')
  }

  const handleReset = () => {
    localStorage.clear()
    sessionStorage.clear()
    router.push('/')
  }

  if (!mounted) {
    return null
  }

  if (loading) {
    return (
      <Box sx={{ 
        height: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        flexDirection: 'column',
        gap: 2,
      }}>
        <CircularProgress size={60} />
        <Typography variant="h6" color="text.secondary">
          AIが企業を分析中...
        </Typography>
      </Box>
    )
  }

  if (error) {
    return (
      <Box sx={{ 
        height: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        flexDirection: 'column',
        gap: 2,
        p: 3,
      }}>
        <Typography variant="h6" color="error">
          {error}
        </Typography>
        <Button variant="outlined" onClick={handleReset}>
          最初からやり直す
        </Button>
      </Box>
    )
  }

  if (companies.length === 0) {
    return (
      <Box sx={{ 
        height: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        flexDirection: 'column',
        gap: 2,
      }}>
        <Typography variant="h6" color="text.secondary">
          適合する企業が見つかりませんでした
        </Typography>
        <Button variant="outlined" onClick={handleReset}>
          最初からやり直す
        </Button>
      </Box>
    )
  }

  return (
    <Box sx={{ 
      height: '100vh',
      display: 'flex',
      flexDirection: 'column',
      overflow: 'hidden',
      backgroundColor: '#fff',
    }}>
      {/* ヘッダー部分 */}
      <Box sx={{ 
        p: 3, 
        borderBottom: '1px solid #e0e0e0',
        backgroundColor: '#fff',
        flexShrink: 0,
      }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
          <Button variant="outlined" startIcon={<ArrowBack />} onClick={handleBack}>
            チャットに戻る
          </Button>
        </Box>
        <Box sx={{ textAlign: 'center' }}>
          <Typography variant="h4" fontWeight="bold" gutterBottom>
            🎉 AI分析完了！適合企業を{companies.length}社に絞り込みました
          </Typography>
          <Typography variant="body1" color="text.secondary">
            AIによる詳細分析に基づいて、最適なIT企業をマッチングしました
          </Typography>
        </Box>
      </Box>

      {/* スクロール可能なコンテンツエリア */}
      <Box sx={{ 
        flexGrow: 1,
        overflowY: 'auto',
        p: 3,
        backgroundColor: '#fafafa',
      }}>
        <Box sx={{ maxWidth: 1200, mx: 'auto' }}>
          <Stack spacing={3}>
            {companies.map((company, index) => (
              <Card key={company.id} elevation={3} sx={{ border: '2px solid', borderColor: 'primary.light' }}>
                <CardContent>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                      <Avatar sx={{ bgcolor: 'primary.main', width: 40, height: 40, fontWeight: 'bold' }}>
                        {index + 1}
                      </Avatar>
                      <Box>
                        <Typography variant="h6" fontWeight="bold">
                          {company.name}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                          {company.industry}
                        </Typography>
                      </Box>
                    </Box>
                    <Box sx={{ textAlign: 'right' }}>
                      <Typography variant="h4" color="primary.main" fontWeight="bold">
                        {company.matchScore}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        適合度
                      </Typography>
                    </Box>
                  </Box>

                  <Typography variant="body2" sx={{ mb: 2 }}>
                    {company.description}
                  </Typography>

                  <Stack direction="row" spacing={2} sx={{ mb: 2, flexWrap: 'wrap' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <LocationOn fontSize="small" color="action" />
                      <Typography variant="body2" color="text.secondary">
                        {company.location}
                      </Typography>
                    </Box>
                    {company.employees && (
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <People fontSize="small" color="action" />
                        <Typography variant="body2" color="text.secondary">
                          {company.employees}
                        </Typography>
                      </Box>
                    )}
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <TrendingUpIcon fontSize="small" color="action" />
                      <Typography variant="body2" color="text.secondary">
                        {company.industry}
                      </Typography>
                    </Box>
                  </Stack>

                  {company.techStack && company.techStack.length > 0 && (
                    <Box sx={{ mb: 2 }}>
                      <Typography variant="caption" color="text.secondary" display="block" gutterBottom>
                        技術スタック:
                      </Typography>
                      <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                        {company.techStack.map((tech, i) => (
                          <Chip key={i} label={tech} size="small" color="primary" variant="outlined" />
                        ))}
                      </Stack>
                    </Box>
                  )}

                  {company.tags && company.tags.length > 0 && (
                    <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap sx={{ mb: 2 }}>
                      {company.tags.map((tag, i) => (
                        <Chip key={i} label={tag} size="small" />
                      ))}
                    </Stack>
                  )}
                </CardContent>
              </Card>
            ))}
          </Stack>

          <Box sx={{ textAlign: 'center', mt: 4, mb: 4 }}>
            <Button variant="outlined" size="large" startIcon={<Refresh />} onClick={handleReset}>
              最初からやり直す
            </Button>
          </Box>
        </Box>
      </Box>
    </Box>
  )
}

export default function ResultsPage() {
  return (
    <Suspense fallback={
      <Box sx={{ 
        height: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}>
        <CircularProgress size={60} />
      </Box>
    }>
      <ResultsContent />
    </Suspense>
  )
}
