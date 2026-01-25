'use client'

import { useEffect, useMemo, useState } from 'react'
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Divider,
  Grid,
  MenuItem,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { authService } from '@/lib/auth'

type CrawlSource = {
  id: number
  name: string
  target_type: string
  source_type?: string
  source_url?: string
  schedule_type: string
  schedule_day: number
  schedule_time: string
  is_active: boolean
  last_run_at?: string
  next_run_at?: string
}

type CrawlRun = {
  id: number
  source_id: number
  status: string
  message?: string
  started_at: string
  ended_at?: string
}

const WEEKDAY_OPTIONS = [
  { value: 0, label: '日' },
  { value: 1, label: '月' },
  { value: 2, label: '火' },
  { value: 3, label: '水' },
  { value: 4, label: '木' },
  { value: 5, label: '金' },
  { value: 6, label: '土' },
]

export default function AdminCrawlingPage() {
  const [sources, setSources] = useState<CrawlSource[]>([])
  const [runs, setRuns] = useState<CrawlRun[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const [name, setName] = useState('')
  const [targetType, setTargetType] = useState<'company' | 'popular_companies'>('company')
  const [sourceType, setSourceType] = useState('official')
  const [sourceUrl, setSourceUrl] = useState('')
  const [scheduleType, setScheduleType] = useState<'weekly' | 'monthly'>('weekly')
  const [scheduleDay, setScheduleDay] = useState(1)
  const [scheduleTime, setScheduleTime] = useState('09:00')

  useEffect(() => {
    const user = authService.getStoredUser()
    if (!user?.is_admin) {
      window.location.href = '/'
    }
  }, [])

  const loadSources = async () => {
    setError('')
    const response = await fetch('/api/admin/crawl-sources')
    const data = await response.json()
    if (!response.ok) {
      setError(data?.error || 'クローリング設定の取得に失敗しました')
      return
    }
    setSources(data?.sources || [])
  }

  const loadRuns = async () => {
    const response = await fetch('/api/admin/crawl-runs')
    const data = await response.json()
    if (response.ok) {
      setRuns(data?.runs || [])
    }
  }

  useEffect(() => {
    loadSources()
    loadRuns()
  }, [])

  const handleCreate = async () => {
    setError('')
    setLoading(true)
    const admin = authService.getStoredUser()
    const payload = {
      name,
      target_type: targetType,
      source_type: sourceType,
      source_url: sourceUrl,
      schedule_type: scheduleType,
      schedule_day: scheduleDay,
      schedule_time: scheduleTime,
    }
    const response = await fetch('/api/admin/crawl-sources', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Admin-Email': admin?.email || '',
      },
      body: JSON.stringify(payload),
    })
    const data = await response.json()
    if (!response.ok) {
      setError(data?.error || 'スケジュール作成に失敗しました')
      setLoading(false)
      return
    }
    setName('')
    setTargetType('company')
    setSourceUrl('')
    setScheduleType('weekly')
    setScheduleDay(1)
    setScheduleTime('09:00')
    await loadSources()
    setLoading(false)
  }

  const handleToggleActive = async (source: CrawlSource) => {
    const admin = authService.getStoredUser()
    const response = await fetch(`/api/admin/crawl-sources/${source.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-Admin-Email': admin?.email || '',
      },
      body: JSON.stringify({ is_active: !source.is_active }),
    })
    if (response.ok) {
      await loadSources()
    }
  }

  const handleRun = async (source: CrawlSource) => {
    const admin = authService.getStoredUser()
    const response = await fetch(`/api/admin/crawl-sources/${source.id}/run`, {
      method: 'POST',
      headers: {
        'X-Admin-Email': admin?.email || '',
      },
    })
    if (response.ok) {
      await loadSources()
      await loadRuns()
    }
  }

  const scheduleLabel = useMemo(() => {
    if (scheduleType === 'weekly') {
      const dayLabel = WEEKDAY_OPTIONS.find((d) => d.value === scheduleDay)?.label ?? '月'
      return `毎週${dayLabel} ${scheduleTime}`
    }
    return `毎月${scheduleDay}日 ${scheduleTime}`
  }, [scheduleType, scheduleDay, scheduleTime])

  return (
    <Box sx={{ p: 4, maxWidth: 1100, mx: 'auto' }}>
      <Typography variant="h4" fontWeight="bold" gutterBottom>
        自動クローリング管理
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        週次・月次で企業データを自動更新します。対象URLごとにスケジュールを設定してください。
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} md={5}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                新規スケジュール
              </Typography>
              <Stack spacing={2}>
                <TextField
                  select
                  label="対象タイプ"
                  value={targetType}
                  onChange={(e) => setTargetType(e.target.value as 'company' | 'popular_companies')}
                >
                  <MenuItem value="company">企業単体</MenuItem>
                  <MenuItem value="popular_companies">人気企業一覧</MenuItem>
                </TextField>
                <TextField
                  label={targetType === 'company' ? '企業名' : '設定名'}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                />
                <TextField
                  select
                  label="出典タイプ"
                  value={sourceType}
                  onChange={(e) => setSourceType(e.target.value)}
                >
                  <MenuItem value="official">公式サイト</MenuItem>
                  <MenuItem value="job_site">就活/転職サイト</MenuItem>
                  <MenuItem value="manual">手入力</MenuItem>
                </TextField>
                <TextField
                  label="出典URL"
                  value={sourceUrl}
                  onChange={(e) => setSourceUrl(e.target.value)}
                />
                <TextField
                  select
                  label="頻度"
                  value={scheduleType}
                  onChange={(e) => setScheduleType(e.target.value as 'weekly' | 'monthly')}
                >
                  <MenuItem value="weekly">毎週</MenuItem>
                  <MenuItem value="monthly">毎月</MenuItem>
                </TextField>
                {scheduleType === 'weekly' ? (
                  <TextField
                    select
                    label="曜日"
                    value={scheduleDay}
                    onChange={(e) => setScheduleDay(Number(e.target.value))}
                  >
                    {WEEKDAY_OPTIONS.map((option) => (
                      <MenuItem key={option.value} value={option.value}>
                        {option.label}
                      </MenuItem>
                    ))}
                  </TextField>
                ) : (
                  <TextField
                    type="number"
                    label="日付"
                    value={scheduleDay}
                    onChange={(e) => setScheduleDay(Number(e.target.value))}
                    inputProps={{ min: 1, max: 31 }}
                  />
                )}
                <TextField
                  type="time"
                  label="実行時刻"
                  value={scheduleTime}
                  onChange={(e) => setScheduleTime(e.target.value)}
                  InputLabelProps={{ shrink: true }}
                />
                <Chip label={scheduleLabel} size="small" sx={{ alignSelf: 'flex-start' }} />
                <Button variant="contained" onClick={handleCreate} disabled={loading}>
                  スケジュールを追加
                </Button>
              </Stack>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={7}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                スケジュール一覧
              </Typography>
              <Divider sx={{ mb: 2 }} />
              <Stack spacing={2}>
                {sources.length === 0 && (
                  <Typography variant="body2" color="text.secondary">
                    まだスケジュールが登録されていません。
                  </Typography>
                )}
                {sources.map((source) => (
                  <Box key={source.id} sx={{ border: '1px solid #eee', borderRadius: 1, p: 2 }}>
                    <Stack spacing={1}>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="subtitle1" fontWeight="bold">
                          {source.name}
                        </Typography>
                        <Chip
                          label={source.is_active ? '稼働中' : '停止中'}
                          size="small"
                          color={source.is_active ? 'success' : 'default'}
                        />
                      </Box>
                      <Typography variant="body2" color="text.secondary">
                        {source.schedule_type === 'weekly'
                          ? `毎週${WEEKDAY_OPTIONS.find((d) => d.value === source.schedule_day)?.label ?? '月'} ${source.schedule_time}`
                          : `毎月${source.schedule_day}日 ${source.schedule_time}`}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        次回: {source.next_run_at ? new Date(source.next_run_at).toLocaleString() : '未設定'} / 前回: {source.last_run_at ? new Date(source.last_run_at).toLocaleString() : '未実行'}
                      </Typography>
                      <Stack direction="row" spacing={1}>
                        <Button size="small" variant="outlined" onClick={() => handleRun(source)}>
                          今すぐ実行
                        </Button>
                        <Button size="small" variant="text" onClick={() => handleToggleActive(source)}>
                          {source.is_active ? '停止する' : '再開する'}
                        </Button>
                      </Stack>
                    </Stack>
                  </Box>
                ))}
              </Stack>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Card>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            最近の実行履歴
          </Typography>
          <Divider sx={{ mb: 2 }} />
          <Stack spacing={1}>
            {runs.length === 0 && (
              <Typography variant="body2" color="text.secondary">
                まだ実行履歴がありません。
              </Typography>
            )}
            {runs.map((run) => (
              <Box key={run.id} sx={{ display: 'flex', justifyContent: 'space-between' }}>
                <Typography variant="body2">
                  #{run.id} / source {run.source_id}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {run.status} · {new Date(run.started_at).toLocaleString()}
                </Typography>
              </Box>
            ))}
          </Stack>
        </CardContent>
      </Card>
    </Box>
  )
}
