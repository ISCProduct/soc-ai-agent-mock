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
  const [error, setError] = useState('')
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
    setError('')
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
      setError(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setLoading(false)
    }
  }

  const handleReview = async () => {
    if (!documentId) {
      setError('document_id が未設定です')
      return
    }
    setError('')
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
      setError(err instanceof Error ? err.message : 'Review failed')
    } finally {
      setReviewLoading(false)
    }
  }

  const handleDownload = () => {
    if (!documentId) {
      setError('document_id が未設定です')
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
          {loading && <LinearProgress />}
          {documentId && (
            <Alert severity="success">document_id: {documentId}</Alert>
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
            label="応募職種 (任意)"
            value={jobTitle}
            onChange={(e) => setJobTitle(e.target.value)}
            fullWidth
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
          {reviewLoading && <LinearProgress />}
          {review && (
            <>
              <Divider />
              <Typography variant="subtitle1">スコア: {review.review.score}</Typography>
              <Typography variant="body2" color="text.secondary">
                {review.review.summary}
              </Typography>
              <List dense>
                {review.items.map((item) => (
                  <ListItem key={item.id} alignItems="flex-start">
                    <ListItemText
                      primary={`P${item.page_number} (${item.severity}) - ${item.message}`}
                      secondary={item.suggestion}
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

      {error && (
        <Alert severity="error" sx={{ mt: 3 }}>
          {error}
        </Alert>
      )}
    </Box>
  )
}
