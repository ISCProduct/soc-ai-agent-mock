'use client'

import {
  Box,
  Chip,
  Divider,
  LinearProgress,
  Paper,
  Stack,
  Typography,
} from '@mui/material'
import { InterviewReport } from '@/lib/interview'
import { parseJsonSafe } from '@/lib/interview-utils'

const PRIMARY = '#ec5b13'

const SCORE_LABELS: Record<string, string> = {
  logic: '論理性',
  specificity: '具体性',
  ownership: '主体性',
  communication: 'コミュニケーション力',
  enthusiasm: '積極性・熱意',
}

interface Props {
  report: InterviewReport
  /** dark = dark background (finished screen), light = light background (history page) */
  theme?: 'dark' | 'light'
}

export default function InterviewSummary({ report, theme = 'dark' }: Props) {
  const isDark = theme === 'dark'
  const paperBg = isDark ? '#2d2e31' : '#fff'
  const paperBorder = isDark ? '1px solid rgba(255,255,255,0.08)' : '1px solid #e2e8f0'
  const textPrimary = isDark ? '#e8eaed' : '#0f172a'
  const textSecondary = isDark ? '#bdc1c6' : '#475569'
  const textMuted = isDark ? '#9aa0a6' : '#64748b'

  const scores = parseJsonSafe(report.scores_json) as Record<string, number> | null
  const evidence = parseJsonSafe(report.evidence_json) as Record<string, string> | null
  const strengths = parseJsonSafe(report.strengths_json) as string[] | null
  const improvements = parseJsonSafe(report.improvements_json) as string[] | null

  // Calculate overall score (average of all categories)
  const overallScore = scores
    ? Math.round((Object.values(scores).reduce((s, v) => s + v, 0) / Object.values(scores).length) * 10) / 10
    : null

  return (
    <Stack spacing={2}>
      {/* Summary text */}
      <Paper sx={{ bgcolor: paperBg, border: paperBorder, p: 3, borderRadius: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
          <Typography sx={{ color: PRIMARY, fontWeight: 700 }}>総合評価</Typography>
          {overallScore !== null && (
            <Chip
              label={`${overallScore} / 5`}
              size="small"
              sx={{ bgcolor: PRIMARY, color: '#fff', fontWeight: 700, fontSize: 13 }}
            />
          )}
        </Box>
        <Typography variant="body2" sx={{ color: textSecondary, lineHeight: 1.8, whiteSpace: 'pre-line' }}>
          {report.summary_text || '要約がありません'}
        </Typography>
      </Paper>

      {/* Scores */}
      {scores && (
        <Paper sx={{ bgcolor: paperBg, border: paperBorder, p: 3, borderRadius: 2 }}>
          <Typography sx={{ color: textPrimary, fontWeight: 700, mb: 2 }}>カテゴリ別スコア</Typography>
          <Stack spacing={1.5}>
            {Object.entries(scores).map(([key, value]) => (
              <Box key={key}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 0.5 }}>
                  <Typography variant="body2" sx={{ color: textSecondary }}>
                    {SCORE_LABELS[key] ?? key}
                  </Typography>
                  <Typography variant="body2" sx={{ fontWeight: 700, color: PRIMARY }}>
                    {value} / 5
                  </Typography>
                </Box>
                <LinearProgress
                  variant="determinate"
                  value={value * 20}
                  sx={{
                    height: 6,
                    borderRadius: 3,
                    bgcolor: isDark ? '#3c4043' : '#e2e8f0',
                    '& .MuiLinearProgress-bar': { bgcolor: PRIMARY, borderRadius: 3 },
                  }}
                />
                {evidence?.[key] && (
                  <Typography variant="caption" sx={{ color: textMuted, mt: 0.5, display: 'block', lineHeight: 1.5 }}>
                    「{evidence[key]}」
                  </Typography>
                )}
              </Box>
            ))}
          </Stack>
        </Paper>
      )}

      {/* Strengths & Improvements */}
      {(strengths?.length || improvements?.length) ? (
        <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 2 }}>
          {strengths && strengths.length > 0 && (
            <Paper sx={{ bgcolor: paperBg, border: paperBorder, p: 3, borderRadius: 2 }}>
              <Typography sx={{ color: '#34a853', fontWeight: 700, mb: 1.5 }}>強み</Typography>
              <Stack spacing={1}>
                {strengths.map((s, i) => (
                  <Box key={i} sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
                    <Typography sx={{ color: '#34a853', fontSize: 16, lineHeight: 1.4, flexShrink: 0 }}>✓</Typography>
                    <Typography variant="body2" sx={{ color: textSecondary, lineHeight: 1.6 }}>{s}</Typography>
                  </Box>
                ))}
              </Stack>
            </Paper>
          )}
          {improvements && improvements.length > 0 && (
            <Paper sx={{ bgcolor: paperBg, border: paperBorder, p: 3, borderRadius: 2 }}>
              <Typography sx={{ color: '#fbbc04', fontWeight: 700, mb: 1.5 }}>改善点</Typography>
              <Stack spacing={1}>
                {improvements.map((item, i) => (
                  <Box key={i} sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
                    <Typography sx={{ color: '#fbbc04', fontSize: 16, lineHeight: 1.4, flexShrink: 0 }}>→</Typography>
                    <Typography variant="body2" sx={{ color: textSecondary, lineHeight: 1.6 }}>{item}</Typography>
                  </Box>
                ))}
              </Stack>
            </Paper>
          )}
        </Box>
      ) : null}
    </Stack>
  )
}
