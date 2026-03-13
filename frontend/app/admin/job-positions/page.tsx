'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Divider,
  Stack,
  Typography,
} from '@mui/material'
import { authService } from '@/lib/auth'

type JobPosition = {
  id: number
  company_id: number
  title: string
  description?: string
  employment_type?: string
  work_location?: string
  remote_option?: boolean
  data_status?: string
  company?: { id: number; name: string }
  job_category?: { id: number; name: string }
}

const statusBadge = (status?: string) => {
  if (status === 'published') return <Chip label="公開" color="success" size="small" />
  if (status === 'rejected') return <Chip label="却下" color="error" size="small" />
  return <Chip label="審査中" color="warning" size="small" />
}

export default function AdminJobPositionsPage() {
  useEffect(() => {
    const user = authService.getStoredUser()
    if (!user?.is_admin) {
      window.location.href = '/'
    }
  }, [])

  const [jobPositions, setJobPositions] = useState<JobPosition[]>([])
  const [error, setError] = useState('')
  const [filterStatus, setFilterStatus] = useState<'all' | 'draft' | 'published' | 'rejected'>('all')

  const fetchJobPositions = async () => {
    const res = await fetch('/api/admin/job-positions?limit=100')
    const data = await res.json()
    if (res.ok) setJobPositions(data?.positions || [])
  }

  useEffect(() => {
    fetchJobPositions()
  }, [])

  const handlePublish = async (id: number) => {
    const admin = authService.getStoredUser()
    const res = await fetch(`/api/admin/job-positions/${id}/publish`, {
      method: 'PATCH',
      headers: { 'X-Admin-Email': admin?.email || '' },
    })
    if (!res.ok) {
      const data = await res.json()
      setError(data?.error || '承認に失敗しました')
      return
    }
    fetchJobPositions()
  }

  const handleReject = async (id: number) => {
    const admin = authService.getStoredUser()
    const res = await fetch(`/api/admin/job-positions/${id}/reject`, {
      method: 'PATCH',
      headers: { 'X-Admin-Email': admin?.email || '' },
    })
    if (!res.ok) {
      const data = await res.json()
      setError(data?.error || '却下に失敗しました')
      return
    }
    fetchJobPositions()
  }

  const filtered = jobPositions.filter((p) => {
    if (filterStatus === 'all') return true
    return (p.data_status || 'draft') === filterStatus
  })

  return (
    <Box sx={{ p: 4, maxWidth: 1000, mx: 'auto' }}>
      <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
        <Typography variant="h4" fontWeight="bold">
          求人管理
        </Typography>
        <Stack direction="row" spacing={1}>
          <Button variant="outlined" size="small" component={Link} href="/admin/companies">
            企業管理
          </Button>
          <Button variant="outlined" size="small" component={Link} href="/admin/graduate-employments">
            就職情報管理
          </Button>
        </Stack>
      </Stack>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        クローリングで取得した求人情報を審査・公開します。
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Card>
        <CardContent>
          <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 2 }}>
            <Typography variant="h6">求人一覧</Typography>
            <Stack direction="row" spacing={1}>
              {(['all', 'draft', 'published', 'rejected'] as const).map((s) => (
                <Chip
                  key={s}
                  label={s === 'all' ? 'すべて' : s === 'draft' ? '審査中' : s === 'published' ? '公開' : '却下'}
                  variant={filterStatus === s ? 'filled' : 'outlined'}
                  color={s === 'published' ? 'success' : s === 'rejected' ? 'error' : s === 'draft' ? 'warning' : 'default'}
                  onClick={() => setFilterStatus(s)}
                  clickable
                />
              ))}
            </Stack>
          </Stack>
          <Divider sx={{ mb: 2 }} />
          <Stack spacing={1}>
            {filtered.length === 0 ? (
              <Typography variant="body2" color="text.secondary">
                該当する求人がありません。
              </Typography>
            ) : (
              filtered.map((position) => (
                <Box key={position.id} sx={{ border: '1px solid #eee', borderRadius: 1, p: 2 }}>
                  <Stack direction="row" alignItems="center" justifyContent="space-between">
                    <Box>
                      <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 0.5 }}>
                        <Typography variant="subtitle1" fontWeight="bold">
                          {position.title}
                        </Typography>
                        {statusBadge(position.data_status)}
                      </Stack>
                      <Typography variant="body2" color="text.secondary">
                        {position.company?.name || `企業ID ${position.company_id}`}
                        {position.job_category?.name ? ` / ${position.job_category.name}` : ''}
                        {position.employment_type ? ` / ${position.employment_type}` : ''}
                        {position.work_location ? ` / ${position.work_location}` : ''}
                        {position.remote_option ? ' / リモート可' : ''}
                      </Typography>
                    </Box>
                    {(position.data_status || 'draft') !== 'published' && (
                      <Stack direction="row" spacing={1}>
                        <Button
                          variant="contained"
                          color="success"
                          size="small"
                          onClick={() => handlePublish(position.id)}
                        >
                          承認
                        </Button>
                        {(position.data_status || 'draft') !== 'rejected' && (
                          <Button
                            variant="outlined"
                            color="error"
                            size="small"
                            onClick={() => handleReject(position.id)}
                          >
                            却下
                          </Button>
                        )}
                      </Stack>
                    )}
                  </Stack>
                </Box>
              ))
            )}
          </Stack>
        </CardContent>
      </Card>
    </Box>
  )
}
