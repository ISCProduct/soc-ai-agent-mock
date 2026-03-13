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
  MenuItem,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { authService } from '@/lib/auth'

type Company = {
  id: number
  name: string
  industry?: string
  location?: string
  website_url?: string
  source_type?: string
  source_url?: string
  is_provisional?: boolean
  data_status?: string
  is_active?: boolean
}

const statusBadge = (status?: string) => {
  if (status === 'published') return <Chip label="公開" color="success" size="small" />
  return <Chip label="下書き" color="warning" size="small" />
}

const sourceLabel = (sourceType?: string) => {
  if (sourceType === 'official') return '公式'
  if (sourceType === 'job_site') return 'クローリング'
  return '手動'
}

export default function AdminCompaniesPage() {
  useEffect(() => {
    const user = authService.getStoredUser()
    if (!user?.is_admin) {
      window.location.href = '/'
    }
  }, [])

  const [companies, setCompanies] = useState<Company[]>([])
  const [error, setError] = useState('')
  const [filterStatus, setFilterStatus] = useState<'all' | 'draft' | 'published'>('all')

  const [name, setName] = useState('')
  const [industry, setIndustry] = useState('')
  const [location, setLocation] = useState('')
  const [websiteUrl, setWebsiteUrl] = useState('')
  const [sourceType, setSourceType] = useState('manual')
  const [sourceUrl, setSourceUrl] = useState('')
  const [dataStatus, setDataStatus] = useState('draft')
  const [isProvisional, setIsProvisional] = useState(true)

  const fetchCompanies = async () => {
    setError('')
    const res = await fetch('/api/admin/companies')
    const data = await res.json()
    if (!res.ok) {
      setError(data?.error || '企業一覧の取得に失敗しました')
      return
    }
    setCompanies(data?.companies || [])
  }

  useEffect(() => {
    fetchCompanies()
  }, [])

  const handleCreate = async () => {
    setError('')
    const admin = authService.getStoredUser()
    const payload = {
      name,
      industry,
      location,
      website_url: websiteUrl,
      source_type: sourceType,
      source_url: sourceUrl,
      is_provisional: isProvisional,
      data_status: dataStatus,
    }
    const res = await fetch('/api/admin/companies', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Admin-Email': admin?.email || '',
      },
      body: JSON.stringify(payload),
    })
    const data = await res.json()
    if (!res.ok) {
      setError(data?.error || '企業の作成に失敗しました')
      return
    }
    setName('')
    setIndustry('')
    setLocation('')
    setWebsiteUrl('')
    setSourceUrl('')
    setIsProvisional(true)
    setDataStatus('draft')
    fetchCompanies()
  }

  const handlePublish = async (companyId: number) => {
    const admin = authService.getStoredUser()
    const res = await fetch(`/api/admin/companies/${companyId}/publish`, {
      method: 'PATCH',
      headers: { 'X-Admin-Email': admin?.email || '' },
    })
    if (!res.ok) {
      const data = await res.json()
      setError(data?.error || '承認に失敗しました')
      return
    }
    fetchCompanies()
  }

  const handleReject = async (companyId: number) => {
    const admin = authService.getStoredUser()
    const res = await fetch(`/api/admin/companies/${companyId}/reject`, {
      method: 'PATCH',
      headers: { 'X-Admin-Email': admin?.email || '' },
    })
    if (!res.ok) {
      const data = await res.json()
      setError(data?.error || '却下に失敗しました')
      return
    }
    fetchCompanies()
  }

  const filteredCompanies = companies.filter((c) => {
    if (filterStatus === 'all') return true
    return c.data_status === filterStatus
  })

  return (
    <Box sx={{ p: 4, maxWidth: 1000, mx: 'auto' }}>
      <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
        <Typography variant="h4" fontWeight="bold">
          企業管理
        </Typography>
        <Stack direction="row" spacing={1}>
          <Button variant="outlined" size="small" component={Link} href="/admin/job-positions">
            求人管理
          </Button>
          <Button variant="outlined" size="small" component={Link} href="/admin/graduate-employments">
            就職情報管理
          </Button>
        </Stack>
      </Stack>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        出典付きの企業情報を登録し、暫定/公開を管理します。
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            企業の追加
          </Typography>
          <Stack spacing={2}>
            <TextField label="企業名" value={name} onChange={(e) => setName(e.target.value)} required />
            <TextField label="業種" value={industry} onChange={(e) => setIndustry(e.target.value)} />
            <TextField label="所在地" value={location} onChange={(e) => setLocation(e.target.value)} />
            <TextField label="公式サイトURL" value={websiteUrl} onChange={(e) => setWebsiteUrl(e.target.value)} />
            <TextField select label="出典タイプ" value={sourceType} onChange={(e) => setSourceType(e.target.value)}>
              <MenuItem value="official">公式サイト</MenuItem>
              <MenuItem value="job_site">就活/転職サイト</MenuItem>
              <MenuItem value="manual">手入力</MenuItem>
            </TextField>
            <TextField label="出典URL" value={sourceUrl} onChange={(e) => setSourceUrl(e.target.value)} />
            <TextField select label="ステータス" value={dataStatus} onChange={(e) => setDataStatus(e.target.value)}>
              <MenuItem value="draft">下書き</MenuItem>
              <MenuItem value="published">公開</MenuItem>
            </TextField>
            <TextField
              select
              label="暫定データ"
              value={isProvisional ? 'yes' : 'no'}
              onChange={(e) => setIsProvisional(e.target.value === 'yes')}
            >
              <MenuItem value="yes">暫定</MenuItem>
              <MenuItem value="no">確定</MenuItem>
            </TextField>
            <Button variant="contained" onClick={handleCreate}>
              追加
            </Button>
          </Stack>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 2 }}>
            <Typography variant="h6">企業一覧</Typography>
            <Stack direction="row" spacing={1}>
              {(['all', 'draft', 'published'] as const).map((s) => (
                <Chip
                  key={s}
                  label={s === 'all' ? 'すべて' : s === 'draft' ? '下書き' : '公開'}
                  variant={filterStatus === s ? 'filled' : 'outlined'}
                  color={s === 'published' ? 'success' : s === 'draft' ? 'warning' : 'default'}
                  onClick={() => setFilterStatus(s)}
                  clickable
                />
              ))}
            </Stack>
          </Stack>
          <Divider sx={{ mb: 2 }} />
          <Stack spacing={1}>
            {filteredCompanies.map((company) => (
              <Box key={company.id} sx={{ border: '1px solid #eee', borderRadius: 1, p: 2 }}>
                <Stack direction="row" alignItems="center" justifyContent="space-between">
                  <Box>
                    <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 0.5 }}>
                      <Typography variant="subtitle1" fontWeight="bold">
                        {company.name}
                      </Typography>
                      {statusBadge(company.data_status)}
                      <Chip label={sourceLabel(company.source_type)} size="small" variant="outlined" />
                      {company.is_provisional && (
                        <Chip label="暫定" size="small" color="default" variant="outlined" />
                      )}
                    </Stack>
                    <Typography variant="body2" color="text.secondary">
                      {company.industry || '業種未設定'} / {company.location || '所在地未設定'}
                    </Typography>
                  </Box>
                  {company.data_status !== 'published' && (
                    <Stack direction="row" spacing={1}>
                      <Button
                        variant="contained"
                        color="success"
                        size="small"
                        onClick={() => handlePublish(company.id)}
                      >
                        承認
                      </Button>
                      <Button
                        variant="outlined"
                        color="error"
                        size="small"
                        onClick={() => handleReject(company.id)}
                      >
                        却下
                      </Button>
                    </Stack>
                  )}
                </Stack>
              </Box>
            ))}
          </Stack>
        </CardContent>
      </Card>
    </Box>
  )
}
