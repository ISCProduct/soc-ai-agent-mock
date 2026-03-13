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

type JobCategory = {
  id: number
  name: string
}

type JobPosition = {
  id: number
  company_id: number
  title: string
  description?: string
  job_category_id: number
  job_category?: JobCategory
  min_salary?: number
  max_salary?: number
  employment_type?: string
  work_location?: string
  remote_option?: boolean
  required_skills?: string
  preferred_skills?: string
  company?: Company
}

export default function AdminJobPositionsPage() {
  useEffect(() => {
    const user = authService.getStoredUser()
    if (!user?.is_admin) {
      window.location.href = '/'
    }
  }, [])

  const [companies, setCompanies] = useState<Company[]>([])
  const [jobCategories, setJobCategories] = useState<JobCategory[]>([])
  const [jobPositions, setJobPositions] = useState<JobPosition[]>([])
  const [error, setError] = useState('')

  const [jobCompanyId, setJobCompanyId] = useState('')
  const [jobTitle, setJobTitle] = useState('')
  const [jobCategoryId, setJobCategoryId] = useState('')
  const [jobDescription, setJobDescription] = useState('')
  const [minSalary, setMinSalary] = useState('')
  const [maxSalary, setMaxSalary] = useState('')
  const [employmentType, setEmploymentType] = useState('')
  const [workLocation, setWorkLocation] = useState('')
  const [remoteOption, setRemoteOption] = useState('no')
  const [requiredSkills, setRequiredSkills] = useState('')
  const [preferredSkills, setPreferredSkills] = useState('')

  const fetchJobPositions = async () => {
    const res = await fetch('/api/admin/job-positions?limit=100')
    const data = await res.json()
    if (res.ok) setJobPositions(data?.positions || [])
  }

  useEffect(() => {
    const fetchCompanies = async () => {
      const res = await fetch('/api/admin/companies')
      const data = await res.json()
      if (res.ok) setCompanies(data?.companies || [])
    }
    const fetchJobCategories = async () => {
      const res = await fetch('/api/admin/job-categories')
      const data = await res.json()
      if (res.ok) setJobCategories(data?.job_categories || [])
    }
    fetchCompanies()
    fetchJobCategories()
    fetchJobPositions()
  }, [])

  const handleCreate = async () => {
    setError('')
    const admin = authService.getStoredUser()
    const payload = {
      company_id: Number(jobCompanyId),
      title: jobTitle,
      description: jobDescription,
      job_category_id: Number(jobCategoryId),
      min_salary: minSalary ? Number(minSalary) : 0,
      max_salary: maxSalary ? Number(maxSalary) : 0,
      employment_type: employmentType,
      work_location: workLocation,
      remote_option: remoteOption === 'yes',
      required_skills: requiredSkills,
      preferred_skills: preferredSkills,
    }
    const res = await fetch('/api/admin/job-positions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Admin-Email': admin?.email || '',
      },
      body: JSON.stringify(payload),
    })
    const data = await res.json()
    if (!res.ok) {
      setError(data?.error || '求人の登録に失敗しました')
      return
    }
    setJobCompanyId('')
    setJobTitle('')
    setJobCategoryId('')
    setJobDescription('')
    setMinSalary('')
    setMaxSalary('')
    setEmploymentType('')
    setWorkLocation('')
    setRemoteOption('no')
    setRequiredSkills('')
    setPreferredSkills('')
    fetchJobPositions()
  }

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
        企業に紐づく求人ポジションを登録・確認します。
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            求人の登録
          </Typography>
          <Stack spacing={2}>
            <TextField
              select
              label="企業"
              value={jobCompanyId}
              onChange={(e) => setJobCompanyId(e.target.value)}
            >
              {companies.map((company) => (
                <MenuItem key={company.id} value={company.id}>
                  {company.name}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              label="募集タイトル"
              value={jobTitle}
              onChange={(e) => setJobTitle(e.target.value)}
              required
            />
            <TextField
              select
              label="職種カテゴリ"
              value={jobCategoryId}
              onChange={(e) => setJobCategoryId(e.target.value)}
            >
              {jobCategories.map((category) => (
                <MenuItem key={category.id} value={category.id}>
                  {category.name}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              label="募集内容"
              value={jobDescription}
              onChange={(e) => setJobDescription(e.target.value)}
              multiline
              minRows={3}
            />
            <TextField
              label="最低年収(万円)"
              value={minSalary}
              onChange={(e) => setMinSalary(e.target.value)}
              type="number"
            />
            <TextField
              label="最高年収(万円)"
              value={maxSalary}
              onChange={(e) => setMaxSalary(e.target.value)}
              type="number"
            />
            <TextField
              label="雇用形態"
              value={employmentType}
              onChange={(e) => setEmploymentType(e.target.value)}
            />
            <TextField
              label="勤務地"
              value={workLocation}
              onChange={(e) => setWorkLocation(e.target.value)}
            />
            <TextField
              select
              label="リモート可"
              value={remoteOption}
              onChange={(e) => setRemoteOption(e.target.value)}
            >
              <MenuItem value="no">不可</MenuItem>
              <MenuItem value="yes">可</MenuItem>
            </TextField>
            <TextField
              label="必須スキル"
              value={requiredSkills}
              onChange={(e) => setRequiredSkills(e.target.value)}
              multiline
              minRows={2}
            />
            <TextField
              label="歓迎スキル"
              value={preferredSkills}
              onChange={(e) => setPreferredSkills(e.target.value)}
              multiline
              minRows={2}
            />
            <Button variant="contained" onClick={handleCreate}>
              求人を登録
            </Button>
          </Stack>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            登録済み求人
          </Typography>
          <Divider sx={{ mb: 2 }} />
          <Stack spacing={1}>
            {jobPositions.length === 0 ? (
              <Typography variant="body2" color="text.secondary">
                求人はまだ登録されていません。
              </Typography>
            ) : (
              jobPositions.map((position) => (
                <Box key={position.id} sx={{ border: '1px solid #eee', borderRadius: 1, p: 2 }}>
                  <Typography variant="subtitle2" fontWeight="bold">
                    {position.title}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    {position.company?.name || `企業ID ${position.company_id}`} / {position.job_category?.name || '職種未設定'}
                  </Typography>
                  {position.employment_type && (
                    <Typography variant="caption" color="text.secondary">
                      {position.employment_type}{position.work_location ? ` / ${position.work_location}` : ''}
                      {position.remote_option ? ' / リモート可' : ''}
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
