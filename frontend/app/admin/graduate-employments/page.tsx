'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
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
}

type JobPosition = {
  id: number
  title: string
  company?: Company
}

type GraduateEmployment = {
  id: number
  company_id: number
  job_position_id?: number
  graduate_name?: string
  graduation_year?: number
  school_name?: string
  department?: string
  hired_at?: string
  note?: string
  company?: Company
  job_position?: JobPosition
}

export default function AdminGraduateEmploymentsPage() {
  useEffect(() => {
    const user = authService.getStoredUser()
    if (!user?.is_admin) {
      window.location.href = '/'
    }
  }, [])

  const [companies, setCompanies] = useState<Company[]>([])
  const [jobPositions, setJobPositions] = useState<JobPosition[]>([])
  const [graduateEntries, setGraduateEntries] = useState<GraduateEmployment[]>([])
  const [error, setError] = useState('')

  const [gradCompanyId, setGradCompanyId] = useState('')
  const [gradJobPositionId, setGradJobPositionId] = useState('')
  const [graduateName, setGraduateName] = useState('')
  const [graduationYear, setGraduationYear] = useState('')
  const [schoolName, setSchoolName] = useState('')
  const [department, setDepartment] = useState('')
  const [hiredAt, setHiredAt] = useState('')
  const [employmentNote, setEmploymentNote] = useState('')

  const fetchGraduateEntries = async () => {
    const res = await fetch('/api/admin/graduate-employments?limit=100')
    const data = await res.json()
    if (res.ok) setGraduateEntries(data?.entries || [])
  }

  useEffect(() => {
    const fetchCompanies = async () => {
      const res = await fetch('/api/admin/companies')
      const data = await res.json()
      if (res.ok) setCompanies(data?.companies || [])
    }
    const fetchJobPositions = async () => {
      const res = await fetch('/api/admin/job-positions?limit=100')
      const data = await res.json()
      if (res.ok) setJobPositions(data?.positions || [])
    }
    fetchCompanies()
    fetchJobPositions()
    fetchGraduateEntries()
  }, [])

  const handleCreate = async () => {
    setError('')
    const admin = authService.getStoredUser()
    const payload = {
      company_id: Number(gradCompanyId),
      job_position_id: gradJobPositionId ? Number(gradJobPositionId) : undefined,
      graduate_name: graduateName,
      graduation_year: graduationYear ? Number(graduationYear) : 0,
      school_name: schoolName,
      department,
      hired_at: hiredAt,
      note: employmentNote,
    }
    const res = await fetch('/api/admin/graduate-employments', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Admin-Email': admin?.email || '',
      },
      body: JSON.stringify(payload),
    })
    const data = await res.json()
    if (!res.ok) {
      setError(data?.error || '就職情報の登録に失敗しました')
      return
    }
    setGradCompanyId('')
    setGradJobPositionId('')
    setGraduateName('')
    setGraduationYear('')
    setSchoolName('')
    setDepartment('')
    setHiredAt('')
    setEmploymentNote('')
    fetchGraduateEntries()
  }

  return (
    <Box sx={{ p: 4, maxWidth: 1000, mx: 'auto' }}>
      <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
        <Typography variant="h4" fontWeight="bold">
          卒業生の就職情報管理
        </Typography>
        <Stack direction="row" spacing={1}>
          <Button variant="outlined" size="small" component={Link} href="/admin/companies">
            企業管理
          </Button>
          <Button variant="outlined" size="small" component={Link} href="/admin/job-positions">
            求人管理
          </Button>
        </Stack>
      </Stack>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        卒業生の就職先情報を登録・確認します。
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            就職情報の登録
          </Typography>
          <Stack spacing={2}>
            <TextField
              select
              label="企業"
              value={gradCompanyId}
              onChange={(e) => setGradCompanyId(e.target.value)}
            >
              {companies.map((company) => (
                <MenuItem key={company.id} value={company.id}>
                  {company.name}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              select
              label="応募職種"
              value={gradJobPositionId}
              onChange={(e) => setGradJobPositionId(e.target.value)}
            >
              <MenuItem value="">未設定</MenuItem>
              {jobPositions.map((position) => (
                <MenuItem key={position.id} value={position.id}>
                  {position.title} ({position.company?.name || '企業未設定'})
                </MenuItem>
              ))}
            </TextField>
            <TextField
              label="卒業生氏名"
              value={graduateName}
              onChange={(e) => setGraduateName(e.target.value)}
            />
            <TextField
              label="卒業年度"
              value={graduationYear}
              onChange={(e) => setGraduationYear(e.target.value)}
              type="number"
            />
            <TextField
              label="学校名"
              value={schoolName}
              onChange={(e) => setSchoolName(e.target.value)}
            />
            <TextField
              label="学科/専攻"
              value={department}
              onChange={(e) => setDepartment(e.target.value)}
            />
            <TextField
              label="就職日 (YYYY-MM-DD)"
              value={hiredAt}
              onChange={(e) => setHiredAt(e.target.value)}
            />
            <TextField
              label="メモ"
              value={employmentNote}
              onChange={(e) => setEmploymentNote(e.target.value)}
              multiline
              minRows={2}
            />
            <Button variant="contained" onClick={handleCreate}>
              就職情報を登録
            </Button>
          </Stack>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            就職情報一覧
          </Typography>
          <Divider sx={{ mb: 2 }} />
          <Stack spacing={1}>
            {graduateEntries.length === 0 ? (
              <Typography variant="body2" color="text.secondary">
                まだ就職情報がありません。
              </Typography>
            ) : (
              graduateEntries.map((entry) => (
                <Box key={entry.id} sx={{ border: '1px solid #eee', borderRadius: 1, p: 2 }}>
                  <Typography variant="subtitle2" fontWeight="bold">
                    {entry.company?.name || `企業ID ${entry.company_id}`}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    {entry.graduate_name || '氏名未設定'} / {entry.school_name || '学校未設定'}
                    {entry.graduation_year ? ` (${entry.graduation_year}卒)` : ''}
                  </Typography>
                  {entry.job_position?.title && (
                    <Typography variant="caption" color="text.secondary">
                      職種: {entry.job_position.title}
                    </Typography>
                  )}
                </Box>
              ))
            )}
          </Stack>
        </CardContent>
      </Card>
    </Box>
  )
}
