'use client'

import { useEffect, useState } from 'react'
import {
  Box,
  Button,
  TextField,
  Typography,
  Paper,
  Stack,
  MenuItem,
  LinearProgress,
  Alert,
  Divider,
  List,
  ListItem,
  ListItemText,
  Chip,
} from '@mui/material'
import { authService } from '@/lib/auth'

type ReviewItem = {
  id: number
  page_number: number
  severity: string
  message: string
  suggestion?: string
}

type ReviewResult = {
  review: {
    id: number
    score: number
    summary: string
  }
  items: ReviewItem[]
}

export default function ResumePage() {
  const [userId, setUserId] = useState('')
  const [sessionId, setSessionId] = useState('')
  const [sourceType, setSourceType] = useState('pdf')
  const [sourceUrl, setSourceUrl] = useState('')
  const [companyName, setCompanyName] = useState('')
  const [jobTitle, setJobTitle] = useState('')
  const [candidateType, setCandidateType] = useState('new_grad')
  const [file, setFile] = useState<File | null>(null)
  const [documentId, setDocumentId] = useState<number | null>(null)
  const [loading, setLoading] = useState(false)
  const [reviewLoading, setReviewLoading] = useState(false)
  const [uploadError, setUploadError] = useState('')
  const [reviewError, setReviewError] = useState('')
  const [review, setReview] = useState<ReviewResult | null>(null)

  useEffect(() => {
    const user = authService.getStoredUser()
    if (user?.user_id) {
      setUserId(String(user.user_id))
    }
    if (typeof window !== 'undefined') {
      const storedSession =
        localStorage.getItem('chat_session_id') ||
        sessionStorage.getItem('chatSessionId') ||
        localStorage.getItem('currentSessionId') ||
        ''
      setSessionId(storedSession)
    }
  }, [])

  const handleUpload = async () => {
    setUploadError('')
    setReview(null)
    setLoading(true)
    try {
      if (!userId) {
        throw new Error('user_id が取得できません。ログインしてください。')
      }
      const formData = new FormData()
      formData.append('user_id', userId)
      if (sessionId) {
        formData.append('session_id', sessionId)
      }
      formData.append('source_type', sourceType)
      if (sourceUrl) {
        formData.append('source_url', sourceUrl)
      }
      if (file) {
        formData.append('file', file)
      }

      const response = await fetch('/api/resume/upload', {
        method: 'POST',
        body: formData,
      })
      if (!response.ok) {
        const errText = await response.text()
        let message = errText
        try {
          const parsed = JSON.parse(errText)
          message = parsed?.error || parsed?.message || errText
        } catch {
          message = errText || 'Upload failed'
        }
        throw new Error(message)
      }
      const data = await response.json()
      setDocumentId(data?.document?.id ?? null)
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setLoading(false)
    }
  }

  const handleReview = async () => {
    if (!documentId) {
      setReviewError('document_id が未設定です')
      return
    }
    if (!companyName.trim() && !jobTitle.trim()) {
      setReviewError('企業名が未入力の場合は応募職種を入力してください')
      return
    }
    setReviewError('')
    setReviewLoading(true)
    try {
      const response = await fetch(`/api/resume/review?document_id=${documentId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          company_name: companyName,
          job_title: jobTitle,
          candidate_type: candidateType,
        }),
      })
      if (!response.ok) {
        const errText = await response.text()
        let message = errText
        try {
          const parsed = JSON.parse(errText)
          message = parsed?.error || parsed?.message || errText
        } catch {
          message = errText || 'Review failed'
        }
        throw new Error(message)
      }
      const data = await response.json()
      if (!data || !data.review) {
        throw new Error('Review response is empty')
      }
      setReview(data)
    } catch (err) {
      setReviewError(err instanceof Error ? err.message : 'Review failed')
    } finally {
      setReviewLoading(false)
    }
  }

  const severityColor = (severity: string): 'error' | 'warning' | 'info' | 'default' => {
    switch (severity) {
      case 'critical': return 'error'
      case 'warning': return 'warning'
      case 'info': return 'info'
      default: return 'default'
    }
  }

  const severityLabel = (severity: string): string => {
    switch (severity) {
      case 'critical': return '重大'
      case 'warning': return '注意'
      case 'info': return '情報'
      default: return severity
    }
  }

  const handleDownload = () => {
    if (!documentId) {
      setReviewError('document_id が未設定です')
      return
    }
    window.open(`/api/resume/annotated?document_id=${documentId}`, '_blank')
  }

  return (
    <Box sx={{ p: 4, maxWidth: 900, mx: 'auto' }}>
      <Typography variant="h4" fontWeight="bold" gutterBottom>
        履歴書・エントリシート レビュー
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
        PDF/DOCX/Google Docsをアップロードして、注釈付きPDFを生成します。
      </Typography>

      <Paper sx={{ p: 3, mb: 3 }} elevation={2}>
        <Stack spacing={2}>
          <TextField
            select
            label="提出形式"
            value={sourceType}
            onChange={(e) => setSourceType(e.target.value)}
            fullWidth
          >
            <MenuItem value="pdf">PDF</MenuItem>
            <MenuItem value="docx">DOCX</MenuItem>
            <MenuItem value="google_docs">Google Docs</MenuItem>
          </TextField>
          <TextField
            label="Google Docs / URL (任意)"
            value={sourceUrl}
            onChange={(e) => setSourceUrl(e.target.value)}
            placeholder="https://... (PDFエクスポートURL)"
            fullWidth
          />
          <Button variant="outlined" component="label">
            ファイルを選択
            <input
              type="file"
              hidden
              accept=".pdf,.docx"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            />
          </Button>
          {file && (
            <Typography variant="body2" color="text.secondary">
              選択ファイル: {file.name}
            </Typography>
          )}
          <Button variant="contained" onClick={handleUpload} disabled={loading}>
            アップロード
          </Button>
          {loading && (
            <Box>
              <LinearProgress />
              <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                アップロード中...
              </Typography>
            </Box>
          )}
          {uploadError && (
            <Alert severity="error">{uploadError}</Alert>
          )}
          {documentId && (
            <Alert severity="success">
              アップロード完了！下のフォームでレビューを実行してください。
            </Alert>
          )}
        </Stack>
      </Paper>

      <Paper sx={{ p: 3 }} elevation={2}>
        <Stack spacing={2}>
          <Typography variant="h6">レビュー実行</Typography>
          <TextField
            label="応募企業名 (任意)"
            value={companyName}
            onChange={(e) => setCompanyName(e.target.value)}
            fullWidth
          />
          <TextField
            label={companyName.trim() ? '応募職種 (任意)' : '応募職種 (企業名未入力の場合は必須)'}
            value={jobTitle}
            onChange={(e) => setJobTitle(e.target.value)}
            fullWidth
            required={!companyName.trim()}
            error={!companyName.trim() && !jobTitle.trim()}
            helperText={!companyName.trim() ? '企業名が未入力の場合は職種を入力するとAIレビューが実行されます' : ''}
          />
          <TextField
            select
            label="候補者区分"
            value={candidateType}
            onChange={(e) => setCandidateType(e.target.value)}
            fullWidth
          >
            <MenuItem value="new_grad">新卒</MenuItem>
            <MenuItem value="mid_career">中途</MenuItem>
          </TextField>
          <Button variant="contained" onClick={handleReview} disabled={reviewLoading || !documentId}>
            レビューを生成
          </Button>
          {reviewLoading && (
            <Box>
              <LinearProgress />
              <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                レビューを生成中...（通常30〜60秒かかります）
              </Typography>
            </Box>
          )}
          {reviewError && (
            <Alert severity="error">{reviewError}</Alert>
          )}
          {review && (
            <>
              <Divider />
              <Typography variant="subtitle1">スコア: {review.review.score}</Typography>
              <Typography variant="body2" color="text.secondary">
                {review.review.summary}
              </Typography>
              <List dense>
                {review.items.map((item) => (
                  <ListItem key={item.id} alignItems="flex-start" sx={{ gap: 1 }}>
                    <Chip
                      label={severityLabel(item.severity)}
                      color={severityColor(item.severity)}
                      size="small"
                      sx={{ mt: 0.5, flexShrink: 0 }}
                    />
                    <ListItemText
                      primary={`P${item.page_number} - ${item.message}`}
                      secondary={item.suggestion ? `改善案: ${item.suggestion}` : undefined}
                    />
                  </ListItem>
                ))}
              </List>
              <Button variant="outlined" onClick={handleDownload}>
                注釈PDFをダウンロード
              </Button>
            </>
          )}
        </Stack>
      </Paper>
    </Box>
  )
}
