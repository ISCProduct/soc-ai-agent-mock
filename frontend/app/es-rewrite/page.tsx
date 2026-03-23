'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Divider,
  IconButton,
  Paper,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import AutoFixHighIcon from '@mui/icons-material/AutoFixHigh'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import CheckIcon from '@mui/icons-material/Check'
import EditNoteIcon from '@mui/icons-material/EditNote'

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:80'
const PRIMARY = '#ec5b13'

const QUESTION_TYPES = ['志望動機', '自己PR', '学チカ', 'ガクチカ', 'その他']

type StarBreakdown = {
  situation: string
  task: string
  action: string
  result: string
}

type RewriteResult = {
  rewritten_text: string
  star: StarBreakdown
}

const STAR_LABELS: { key: keyof StarBreakdown; label: string; color: string; emoji: string }[] = [
  { key: 'situation', label: 'Situation（状況）', color: '#3b82f6', emoji: '📍' },
  { key: 'task',      label: 'Task（課題）',      color: '#8b5cf6', emoji: '🎯' },
  { key: 'action',    label: 'Action（施策）',    color: PRIMARY,   emoji: '⚡' },
  { key: 'result',    label: 'Result（成果）',    color: '#10b981', emoji: '📊' },
]

export default function ESRewritePage() {
  const router = useRouter()

  const [originalText, setOriginalText] = useState('')
  const [questionType, setQuestionType] = useState('学チカ')
  const [techStack, setTechStack] = useState('')
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<RewriteResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const handleRewrite = async () => {
    if (!originalText.trim()) return
    setLoading(true)
    setError(null)
    setResult(null)

    try {
      const res = await fetch(`${BACKEND_URL}/api/es/rewrite`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          original_text: originalText,
          question_type: questionType,
          tech_stack: techStack,
        }),
      })
      if (!res.ok) throw new Error(await res.text())
      const data: RewriteResult = await res.json()
      setResult(data)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'リライトに失敗しました。再試行してください。')
    } finally {
      setLoading(false)
    }
  }

  const handleCopy = async () => {
    if (!result) return
    await navigator.clipboard.writeText(result.rewritten_text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: '#f8f6f6' }}>
      {/* Header */}
      <Box
        component="header"
        sx={{
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          px: { xs: 3, lg: 8 }, py: 2,
          bgcolor: '#fff', borderBottom: '1px solid #e2e8f0',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Box sx={{ color: PRIMARY }}><EditNoteIcon sx={{ fontSize: 32 }} /></Box>
          <Box>
            <Typography sx={{ fontWeight: 700, fontSize: 20, color: '#0f172a' }}>ESリライト</Typography>
            <Typography sx={{ fontSize: 12, color: '#64748b' }}>AIがあなたのES文章をエンジニア向けにリライトします</Typography>
          </Box>
        </Box>
        <IconButton onClick={() => router.push('/')} sx={{ bgcolor: '#f1f5f9', color: '#475569' }}>
          <ArrowBackIcon />
        </IconButton>
      </Box>

      {/* Main */}
      <Box sx={{ maxWidth: 1200, mx: 'auto', p: { xs: 2, md: 4 } }}>
        <Box
          sx={{
            display: 'grid',
            gridTemplateColumns: { xs: '1fr', lg: result ? '1fr 1fr' : '1fr' },
            gap: 3,
            alignItems: 'start',
          }}
        >
          {/* ── Left: Input ── */}
          <Paper elevation={0} sx={{ p: 3, borderRadius: 2, border: '1px solid #e2e8f0', bgcolor: '#fff' }}>
            <Typography sx={{ fontWeight: 700, fontSize: 17, mb: 2.5 }}>元のES文章を入力</Typography>

            {/* 質問種別 */}
            <Box sx={{ mb: 2.5 }}>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: '#475569', mb: 1 }}>質問種別</Typography>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {QUESTION_TYPES.map(type => (
                  <Chip
                    key={type}
                    label={type}
                    onClick={() => setQuestionType(type)}
                    sx={{
                      cursor: 'pointer',
                      fontWeight: 600,
                      bgcolor: questionType === type ? PRIMARY : '#f1f5f9',
                      color: questionType === type ? '#fff' : '#475569',
                      '&:hover': { bgcolor: questionType === type ? `${PRIMARY}e0` : '#e2e8f0' },
                    }}
                  />
                ))}
              </Box>
            </Box>

            {/* ES本文 */}
            <TextField
              multiline
              rows={10}
              fullWidth
              value={originalText}
              onChange={e => setOriginalText(e.target.value)}
              placeholder={`例）チームで開発した経験があります。最初は上手くいきませんでしたが、話し合いを重ねて最終的には完成させることができました。この経験から協調性の大切さを学びました。`}
              sx={{
                mb: 2.5,
                '& .MuiOutlinedInput-root': {
                  fontSize: 14, lineHeight: 1.8,
                  '&:hover .MuiOutlinedInput-notchedOutline': { borderColor: PRIMARY },
                  '&.Mui-focused .MuiOutlinedInput-notchedOutline': { borderColor: PRIMARY },
                },
              }}
            />

            {/* 技術スタック（任意） */}
            <TextField
              fullWidth
              size="small"
              value={techStack}
              onChange={e => setTechStack(e.target.value)}
              label="使用技術スタック（任意）"
              placeholder="例: React, Node.js, PostgreSQL, Docker"
              sx={{
                mb: 3,
                '& .MuiOutlinedInput-root': {
                  '&:hover .MuiOutlinedInput-notchedOutline': { borderColor: PRIMARY },
                  '&.Mui-focused .MuiOutlinedInput-notchedOutline': { borderColor: PRIMARY },
                },
                '& .MuiInputLabel-root.Mui-focused': { color: PRIMARY },
              }}
            />

            <Button
              variant="contained"
              fullWidth
              size="large"
              disabled={!originalText.trim() || loading}
              onClick={handleRewrite}
              startIcon={loading ? <CircularProgress size={18} sx={{ color: '#fff' }} /> : <AutoFixHighIcon />}
              sx={{
                bgcolor: PRIMARY, '&:hover': { bgcolor: `${PRIMARY}e0` },
                borderRadius: 2, py: 1.5, fontWeight: 700, fontSize: 16,
                textTransform: 'none',
                '&:disabled': { bgcolor: '#e2e8f0', color: '#94a3b8' },
              }}
            >
              {loading ? 'リライト中...' : 'AIでリライトする'}
            </Button>

            {error && (
              <Alert severity="error" sx={{ mt: 2, borderRadius: 2 }}>{error}</Alert>
            )}
          </Paper>

          {/* ── Right: Result ── */}
          {result && (
            <Stack spacing={2}>
              {/* Rewritten text */}
              <Paper elevation={0} sx={{ p: 3, borderRadius: 2, border: `2px solid ${PRIMARY}`, bgcolor: `${PRIMARY}04` }}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <AutoFixHighIcon sx={{ color: PRIMARY, fontSize: 20 }} />
                    <Typography sx={{ fontWeight: 700, fontSize: 17, color: PRIMARY }}>リライト後</Typography>
                  </Box>
                  <Tooltip title={copied ? 'コピーしました' : 'クリップボードにコピー'}>
                    <IconButton
                      size="small"
                      onClick={handleCopy}
                      sx={{ bgcolor: copied ? '#10b981' : '#f1f5f9', '&:hover': { bgcolor: copied ? '#059669' : '#e2e8f0' } }}
                    >
                      {copied
                        ? <CheckIcon sx={{ color: '#fff', fontSize: 18 }} />
                        : <ContentCopyIcon sx={{ color: '#475569', fontSize: 18 }} />
                      }
                    </IconButton>
                  </Tooltip>
                </Box>
                <Typography
                  sx={{ fontSize: 14, lineHeight: 1.9, color: '#1e293b', whiteSpace: 'pre-wrap' }}
                >
                  {result.rewritten_text}
                </Typography>
              </Paper>

              {/* STAR breakdown */}
              <Paper elevation={0} sx={{ p: 3, borderRadius: 2, border: '1px solid #e2e8f0', bgcolor: '#fff' }}>
                <Typography sx={{ fontWeight: 700, fontSize: 16, mb: 2 }}>STAR法 分解</Typography>
                <Stack spacing={2}>
                  {STAR_LABELS.map(({ key, label, color, emoji }, idx) => (
                    <Box key={key}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.8 }}>
                        <Box
                          sx={{
                            width: 28, height: 28, borderRadius: 1.5,
                            bgcolor: `${color}15`,
                            display: 'flex', alignItems: 'center', justifyContent: 'center',
                            fontSize: 14,
                          }}
                        >
                          {emoji}
                        </Box>
                        <Typography sx={{ fontWeight: 700, fontSize: 13, color }}>
                          {label}
                        </Typography>
                      </Box>
                      <Typography
                        variant="body2"
                        sx={{
                          color: '#475569', lineHeight: 1.75,
                          borderLeft: `3px solid ${color}40`, pl: 1.5,
                        }}
                      >
                        {result.star[key] || '—'}
                      </Typography>
                      {idx < STAR_LABELS.length - 1 && (
                        <Divider sx={{ mt: 1.5, borderColor: '#f1f5f9' }} />
                      )}
                    </Box>
                  ))}
                </Stack>
              </Paper>

              {/* Compare hint */}
              <Paper elevation={0} sx={{ p: 2, borderRadius: 2, border: '1px solid #e2e8f0', bgcolor: '#fff' }}>
                <Typography sx={{ fontWeight: 700, fontSize: 13, mb: 1, color: '#64748b' }}>元の文章</Typography>
                <Typography variant="body2" sx={{ color: '#94a3b8', lineHeight: 1.8, whiteSpace: 'pre-wrap' }}>
                  {originalText}
                </Typography>
              </Paper>
            </Stack>
          )}
        </Box>
      </Box>
    </Box>
  )
}
