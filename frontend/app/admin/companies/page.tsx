'use client'

import { useEffect, useState } from 'react'
import {
  Box,
  Button,
  Card,
  CardContent,
  Divider,
  MenuItem,
  Stack,
  TextField,
  Typography,
  Alert,
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

export default function AdminCompaniesPage() {
  useEffect(() => {
    const user = authService.getStoredUser()
    if (!user?.is_admin) {
      window.location.href = '/'
    }
  }, [])

  const [companies, setCompanies] = useState<Company[]>([])
  const [jobCategories, setJobCategories] = useState<JobCategory[]>([])
  const [jobPositions, setJobPositions] = useState<JobPosition[]>([])
  const [graduateEntries, setGraduateEntries] = useState<GraduateEmployment[]>([])
  const [error, setError] = useState('')
  const [name, setName] = useState('')
  const [industry, setIndustry] = useState('')
  const [location, setLocation] = useState('')
  const [websiteUrl, setWebsiteUrl] = useState('')
  const [sourceType, setSourceType] = useState('manual')
  const [sourceUrl, setSourceUrl] = useState('')
  const [dataStatus, setDataStatus] = useState('draft')
  const [isProvisional, setIsProvisional] = useState(true)

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

  const [gradCompanyId, setGradCompanyId] = useState('')
  const [gradJobPositionId, setGradJobPositionId] = useState('')
  const [graduateName, setGraduateName] = useState('')
  const [graduationYear, setGraduationYear] = useState('')
  const [schoolName, setSchoolName] = useState('')
  const [department, setDepartment] = useState('')
  const [hiredAt, setHiredAt] = useState('')
  const [employmentNote, setEmploymentNote] = useState('')

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

  const fetchJobCategories = async () => {
    const res = await fetch('/api/admin/job-categories')
    const data = await res.json()
    if (res.ok) {
      setJobCategories(data?.job_categories || [])
    }
  }

  const fetchJobPositions = async () => {
    const res = await fetch('/api/admin/job-positions?limit=50')
    const data = await res.json()
    if (res.ok) {
      setJobPositions(data?.positions || [])
    }
  }

  const fetchGraduateEntries = async () => {
    const res = await fetch('/api/admin/graduate-employments?limit=50')
    const data = await res.json()
    if (res.ok) {
      setGraduateEntries(data?.entries || [])
    }
  }

  useEffect(() => {
    fetchCompanies()
    fetchJobCategories()
    fetchJobPositions()
    fetchGraduateEntries()
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

  const handleCreateJobPosition = async () => {
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

  const handleCreateGraduateEmployment = async () => {
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
      <Typography variant="h4" fontWeight="bold" gutterBottom>
        管理者: 企業データ管理
      </Typography>
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
            <Button variant="contained" onClick={handleCreateJobPosition}>
              求人を登録
            </Button>
          </Stack>
        </CardContent>
      </Card>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            卒業生の就職情報
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
            <Button variant="contained" onClick={handleCreateGraduateEmployment}>
              就職情報を登録
            </Button>
          </Stack>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            企業一覧
          </Typography>
          <Divider sx={{ mb: 2 }} />
          <Stack spacing={1}>
            {companies.map((company) => (
              <Box key={company.id} sx={{ border: '1px solid #eee', borderRadius: 1, p: 2 }}>
                <Typography variant="subtitle1" fontWeight="bold">
                  {company.name}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {company.industry || '業種未設定'} / {company.location || '所在地未設定'}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {company.data_status || 'draft'} / {company.is_provisional ? '暫定' : '確定'} / {company.source_type || 'manual'}
                </Typography>
              </Box>
            ))}
          </Stack>
        </CardContent>
      </Card>

      <Card sx={{ mt: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            登録済み求人
          </Typography>
          <Divider sx={{ mb: 2 }} />
          <Stack spacing={1}>
            {jobPositions.length === 0 && (
              <Typography variant="body2" color="text.secondary">
                求人はまだ登録されていません。
              </Typography>
            )}
            {jobPositions.map((position) => (
              <Box key={position.id} sx={{ border: '1px solid #eee', borderRadius: 1, p: 2 }}>
                <Typography variant="subtitle2" fontWeight="bold">
                  {position.title}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {position.company?.name || `企業ID ${position.company_id}`} / {position.job_category?.name || '職種未設定'}
                </Typography>
              </Box>
            ))}
          </Stack>
        </CardContent>
      </Card>

      <Card sx={{ mt: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            卒業生の就職情報一覧
          </Typography>
          <Divider sx={{ mb: 2 }} />
          <Stack spacing={1}>
            {graduateEntries.length === 0 && (
              <Typography variant="body2" color="text.secondary">
                まだ就職情報がありません。
              </Typography>
            )}
            {graduateEntries.map((entry) => (
              <Box key={entry.id} sx={{ border: '1px solid #eee', borderRadius: 1, p: 2 }}>
                <Typography variant="subtitle2" fontWeight="bold">
                  {entry.company?.name || `企業ID ${entry.company_id}`}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {entry.graduate_name || '氏名未設定'} / {entry.school_name || '学校未設定'} {entry.graduation_year ? `(${entry.graduation_year}卒)` : ''}
                </Typography>
                {entry.job_position?.title && (
                  <Typography variant="caption" color="text.secondary">
                    職種: {entry.job_position.title}
                  </Typography>
                )}
              </Box>
            ))}
          </Stack>
        </CardContent>
      </Card>
    </Box>
  )
}
